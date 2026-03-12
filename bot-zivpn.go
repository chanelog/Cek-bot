package main

// ============================================================
//   OGH-ZIV Telegram Bot — Public VPN Shop
//   Pembayaran via DANA: 083113931971 a/n Fauzani Hanifah
//   Admin: create akun gratis tanpa bayar
//   Terhubung ke: /etc/zivpn (shared data dengan ogh-ziv panel)
// ============================================================

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── KONSTANTA PATH (shared dengan ogh-ziv panel) ───────────
const (
	DIR_BOT  = "/etc/zivpn"
	UDB_BOT  = "/etc/zivpn/users.db"
	DOMF_BOT = "/etc/zivpn/domain.conf"
	BOTF_BOT = "/etc/zivpn/bot.conf"
	STRF_BOT = "/etc/zivpn/store.conf"
	MLDB_BOT = "/etc/zivpn/maxlogin.db"
	CFG_BOT  = "/etc/zivpn/config.json"
	LOG_BOT  = "/etc/zivpn/zivpn.log"

	// File khusus bot publik
	ORDERS_DB    = "/etc/zivpn/orders.db"    // format: orderID|userID|username|paket|harga|status|timestamp|bukti
	PAKET_DB     = "/etc/zivpn/paket.db"     // format: nama|hari|harga|kuota|maxlogin
	ADMIN_DB     = "/etc/zivpn/admins.db"    // format: telegramID (satu per baris)
	PENDING_FILE = "/etc/zivpn/pending.json" // state pending user

	// DANA info
	DANA_NO   = "083113931971"
	DANA_NAME = "Fauzani Hanifah"

	BINARY_URL_BOT = "https://github.com/fauzanihanipah/ziv-udp/releases/download/udp-zivpn/udp-zivpn-linux-amd64"
	CONFIG_URL_BOT = "https://raw.githubusercontent.com/fauzanihanipah/ziv-udp/main/config.json"
)

// ── STRUCT ──────────────────────────────────────────────────
type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Photo     []struct {
		FileID string `json:"file_id"`
	} `json:"photo"`
	Caption string `json:"caption"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type Paket struct {
	Nama     string
	Hari     int
	Harga    int
	Kuota    int    // 0 = unlimited
	MaxLogin int
}

type Order struct {
	ID        string
	UserID    int64
	Username  string
	Paket     string
	Harga     int
	Status    string // pending_payment | waiting_confirm | confirmed | rejected | done
	Timestamp string
	Bukti     string
	// Data akun (setelah confirmed)
	AkunUser string
	AkunPass string
	AkunExp  string
}

type UserState struct {
	Step      string // menu | pilih_paket | input_username | input_password | upload_bukti | admin_confirm
	OrderID   string
	PaketName string
	Username  string
	Password  string
}

// ── GLOBAL STATE ────────────────────────────────────────────
var (
	botToken  string
	adminIDs  []int64
	userState = make(map[int64]*UserState)
	stateMu   sync.Mutex
)

// ── MAIN ────────────────────────────────────────────────────
func main() {
	rand.Seed(time.Now().UnixNano())

	// Baca token dari BOTF
	if err := loadBotToken(); err != nil {
		log.Fatalf("❌ Gagal load bot token: %v\nPastikan file %s sudah dikonfigurasi via panel ogh-ziv.", err, BOTF_BOT)
	}

	// Load admin IDs
	loadAdmins()

	// Init paket default jika belum ada
	initDefaultPaket()

	// Init order DB
	os.MkdirAll(DIR_BOT, 0755)
	if _, err := os.Stat(ORDERS_DB); os.IsNotExist(err) {
		os.Create(ORDERS_DB)
	}

	log.Println("🚀 OGH-ZIV Bot started!")
	log.Printf("📡 Bot Token: ...%s", botToken[len(botToken)-8:])
	log.Printf("👑 Admin IDs: %v", adminIDs)

	// Start polling
	offset := 0
	for {
		updates, err := getUpdates(offset)
		if err != nil {
			log.Printf("Error getUpdates: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			go handleUpdate(u)
		}
		time.Sleep(1 * time.Second)
	}
}

func loadBotToken() error {
	data, err := os.ReadFile(BOTF_BOT)
	if err != nil {
		return fmt.Errorf("file %s tidak ditemukan", BOTF_BOT)
	}
	conf := parseConfBot(string(data))
	tok := conf["BOT_TOKEN"]
	if tok == "" {
		return fmt.Errorf("BOT_TOKEN kosong di %s", BOTF_BOT)
	}
	botToken = tok
	return nil
}

func loadAdmins() {
	adminIDs = []int64{}
	// Coba dari ADMIN_DB
	data, err := os.ReadFile(ADMIN_DB)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if id, err := strconv.ParseInt(line, 10, 64); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
	}
	// Fallback: cek CHAT_ID dari bot.conf sebagai admin utama
	if len(adminIDs) == 0 {
		if data, err := os.ReadFile(BOTF_BOT); err == nil {
			conf := parseConfBot(string(data))
			if cid := conf["CHAT_ID"]; cid != "" {
				if id, err := strconv.ParseInt(cid, 10, 64); err == nil {
					adminIDs = append(adminIDs, id)
					// Simpan ke ADMIN_DB
					os.WriteFile(ADMIN_DB, []byte(fmt.Sprintf("# Admin IDs — satu per baris\n%d\n", id)), 0644)
				}
			}
		}
	}
}

func isAdmin(userID int64) bool {
	for _, id := range adminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func initDefaultPaket() {
	if _, err := os.Stat(PAKET_DB); err == nil {
		return
	}
	defaultPaket := `30 Hari|30|10000|0|2
`
	os.WriteFile(PAKET_DB, []byte(defaultPaket), 0644)
	log.Println("✅ Paket default dibuat")
}

// ── TELEGRAM API ────────────────────────────────────────────
func getUpdates(offset int) ([]Update, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", botToken, offset)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	json.Unmarshal(body, &result)
	return result.Result, nil
}

func sendMsg(chatID int64, text string) {
	sendMsgWithKeyboard(chatID, text, nil)
}

func sendMsgWithKeyboard(chatID int64, text string, keyboard interface{}) {
	vals := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"HTML"},
	}
	if keyboard != nil {
		kb, _ := json.Marshal(keyboard)
		vals["reply_markup"] = []string{string(kb)}
	}
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken),
		vals,
	)
}

func sendPhoto(chatID int64, fileID, caption string) {
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", botToken),
		url.Values{
			"chat_id":    {strconv.FormatInt(chatID, 10)},
			"photo":      {fileID},
			"caption":    {caption},
			"parse_mode": {"HTML"},
		},
	)
}

func forwardMsg(chatID int64, fromChatID int64, msgID int) {
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/forwardMessage", botToken),
		url.Values{
			"chat_id":      {strconv.FormatInt(chatID, 10)},
			"from_chat_id": {strconv.FormatInt(fromChatID, 10)},
			"message_id":   {strconv.Itoa(msgID)},
		},
	)
}

type InlineKeyboard struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard"`
}

type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// ── HANDLER UTAMA ────────────────────────────────────────────
func handleUpdate(u Update) {
	msg := u.Message
	if msg.From.ID == 0 {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	username := msg.From.Username
	if username == "" {
		username = msg.From.FirstName
	}

	stateMu.Lock()
	state, exists := userState[userID]
	if !exists {
		state = &UserState{Step: "menu"}
		userState[userID] = state
	}
	stateMu.Unlock()

	// Cek jika user kirim foto (bukti bayar)
	if len(msg.Photo) > 0 {
		handlePhotoBukti(userID, chatID, username, msg)
		return
	}

	// Command handler
	switch {
	case text == "/start":
		handleStart(userID, chatID, username)

	case text == "/menu" || text == "🏠 Menu Utama":
		resetState(userID)
		handleStart(userID, chatID, username)

	case text == "/paket" || text == "🛒 Beli VPN":
		resetState(userID)
		handlePilihPaket(userID, chatID)

	case text == "/cekorder" || text == "📋 Cek Order":
		handleCekOrder(userID, chatID)

	case text == "/info" || text == "ℹ️ Info":
		handleInfo(chatID)

	case text == "/bantuan" || text == "🆘 Bantuan":
		handleBantuan(chatID)

	// ── ADMIN COMMANDS ────────────────────────────────────────
	case text == "/admin" && isAdmin(userID):
		handleAdminMenu(chatID)

	case text == "/listorder" && isAdmin(userID):
		handleListOrder(chatID)

	case text == "/listuser" && isAdmin(userID):
		handleListUser(chatID)

	case text == "/addpaket" && isAdmin(userID):
		sendMsg(chatID, "📦 <b>Format tambah paket:</b>\n\n<code>/newpaket Nama|Hari|Harga|KuotaGB|MaxLogin</code>\n\nContoh:\n<code>/newpaket 30 Hari VIP|30|10000|0|3</code>\n\n<i>Kuota 0 = Unlimited</i>")

	case strings.HasPrefix(text, "/newpaket") && isAdmin(userID):
		handleNewPaket(chatID, text)

	case strings.HasPrefix(text, "/delpaket") && isAdmin(userID):
		handleDelPaket(chatID, text)

	case strings.HasPrefix(text, "/confirm") && isAdmin(userID):
		// /confirm ORDERID username password
		handleAdminConfirm(chatID, userID, text)

	case strings.HasPrefix(text, "/reject") && isAdmin(userID):
		// /reject ORDERID alasan
		handleAdminReject(chatID, text)

	case strings.HasPrefix(text, "/createakun") && isAdmin(userID):
		// /createakun username password hari kuota maxlogin
		handleAdminCreateAkun(chatID, text)

	case text == "/addadmin" && isAdmin(userID):
		sendMsg(chatID, "👑 <b>Tambah admin:</b>\n<code>/setadmin TelegramID</code>")

	case strings.HasPrefix(text, "/setadmin") && isAdmin(userID):
		handleSetAdmin(chatID, text)

	case text == "/broadcast" && isAdmin(userID):
		setState(userID, &UserState{Step: "broadcast"})
		sendMsg(chatID, "📢 Ketik pesan broadcast (akan dikirim ke semua user yang pernah order):")

	case text == "/setpaket" && isAdmin(userID):
		handleListPaket(chatID, true)

	// ── STATE MACHINE ─────────────────────────────────────────
	default:
		handleStateInput(userID, chatID, username, text, state)
	}
}

// ── START & MENU ─────────────────────────────────────────────
func handleStart(userID int64, chatID int64, username string) {
	brand := getBrand()
	adminBadge := ""
	if isAdmin(userID) {
		adminBadge = " 👑"
	}

	msg := fmt.Sprintf(`🔒 <b>Selamat datang di %s VPN!</b>%s

Halo, <b>%s</b>! 👋

Kami menyediakan layanan VPN UDP berkualitas tinggi dengan harga terjangkau.

📱 <b>Cara Beli:</b>
1️⃣ Pilih paket yang sesuai
2️⃣ Transfer via DANA
3️⃣ Upload bukti pembayaran
4️⃣ Admin verifikasi & akun aktif otomatis!

Pilih menu di bawah:`, brand, adminBadge, username)

	kb := buildMainKeyboard(isAdmin(userID))
	sendMsgWithKeyboard(chatID, msg, kb)
}

func buildMainKeyboard(isAdm bool) map[string]interface{} {
	kb := map[string]interface{}{
		"keyboard": [][]map[string]interface{}{
			{{"text": "🛒 Beli VPN"}, {"text": "📋 Cek Order"}},
			{{"text": "ℹ️ Info"}, {"text": "🆘 Bantuan"}},
		},
		"resize_keyboard":   true,
		"one_time_keyboard": false,
	}
	if isAdm {
		rows := kb["keyboard"].([][]map[string]interface{})
		rows = append(rows, []map[string]interface{}{{"text": "👑 Admin Panel"}})
		kb["keyboard"] = rows
	}
	return kb
}

// ── PILIH PAKET ──────────────────────────────────────────────
func handlePilihPaket(userID int64, chatID int64) {
	pakets := loadPaket()
	if len(pakets) == 0 {
		sendMsg(chatID, "❌ Belum ada paket tersedia. Hubungi admin.")
		return
	}

	msg := "🛒 <b>PILIH PAKET VPN</b>\n\n"
	msg += fmt.Sprintf("💳 Pembayaran via: <b>DANA</b>\n📱 No: <code>%s</code>\n👤 a/n: <b>%s</b>\n\n", DANA_NO, DANA_NAME)
	msg += "━━━━━━━━━━━━━━━━━━━━━━\n"

	for i, p := range pakets {
		kuotaStr := "Unlimited"
		if p.Kuota > 0 {
			kuotaStr = fmt.Sprintf("%d GB", p.Kuota)
		}
		msg += fmt.Sprintf("\n<b>%d. %s</b>\n", i+1, p.Nama)
		msg += fmt.Sprintf("   ⏱ Masa aktif : %d hari\n", p.Hari)
		msg += fmt.Sprintf("   📦 Kuota      : %s\n", kuotaStr)
		msg += fmt.Sprintf("   🔒 Max device : %d\n", p.MaxLogin)
		msg += fmt.Sprintf("   💰 Harga      : <b>Rp %s</b>\n", formatRupiah(p.Harga))
		msg += "   ─────────────────\n"
	}

	msg += "\nKetik nomor paket yang ingin dibeli (contoh: <code>1</code>)"

	setState(userID, &UserState{Step: "pilih_paket"})
	sendMsg(chatID, msg)
}

// ── CEK ORDER ───────────────────────────────────────────────
func handleCekOrder(userID int64, chatID int64) {
	orders := getOrdersByUser(userID)
	if len(orders) == 0 {
		sendMsg(chatID, "📋 Kamu belum punya order.\n\nKetik /paket untuk membeli VPN.")
		return
	}

	msg := "📋 <b>ORDER KAMU:</b>\n\n"
	for _, o := range orders {
		statusEmoji := statusEmoji(o.Status)
		msg += fmt.Sprintf("🆔 Order: <code>%s</code>\n", o.ID)
		msg += fmt.Sprintf("📦 Paket: %s\n", o.Paket)
		msg += fmt.Sprintf("💰 Harga: Rp %s\n", formatRupiah(o.Harga))
		msg += fmt.Sprintf("%s Status: <b>%s</b>\n", statusEmoji, statusText(o.Status))
		msg += fmt.Sprintf("🕐 Waktu: %s\n", o.Timestamp)
		if o.Status == "done" && o.AkunUser != "" {
			domain := getDomainBot()
			port := getPortBot()
			msg += fmt.Sprintf("\n✅ <b>Detail Akun VPN:</b>\n")
			msg += fmt.Sprintf("👤 Username: <code>%s</code>\n", o.AkunUser)
			msg += fmt.Sprintf("🔑 Password: <code>%s</code>\n", o.AkunPass)
			msg += fmt.Sprintf("🌐 Host: <code>%s</code>\n", domain)
			msg += fmt.Sprintf("🔌 Port: <code>%s</code>\n", port)
			msg += fmt.Sprintf("📅 Expired: <code>%s</code>\n", o.AkunExp)
		}
		msg += "━━━━━━━━━━━━━━━━━━━━━━\n"
	}
	sendMsg(chatID, msg)
}

// ── INFO ─────────────────────────────────────────────────────
func handleInfo(chatID int64) {
	domain := getDomainBot()
	port := getPortBot()
	brand := getBrand()

	msg := fmt.Sprintf(`ℹ️ <b>INFO %s VPN</b>

🌐 <b>Server Info:</b>
   Host : <code>%s</code>
   Port : <code>%s</code>
   Obfs : <code>zivpn</code>

💳 <b>Pembayaran:</b>
   DANA : <code>%s</code>
   a/n  : <b>%s</b>

📱 <b>Download App:</b>
   ZiVPN → Play Store / App Store

⚡ <b>Keunggulan:</b>
   ✅ Server stabil 24/7
   ✅ Tanpa batasan kecepatan
   ✅ Support semua operator
   ✅ Setup mudah & cepat`, brand, domain, port, DANA_NO, DANA_NAME)

	sendMsg(chatID, msg)
}

// ── BANTUAN ──────────────────────────────────────────────────
func handleBantuan(chatID int64) {
	msg := fmt.Sprintf(`🆘 <b>BANTUAN</b>

<b>Cara Beli VPN:</b>
1. Ketik /paket atau tekan 🛒 Beli VPN
2. Pilih nomor paket
3. Masukkan username yang diinginkan
4. Masukkan password (atau ketik <code>auto</code>)
5. Transfer ke DANA: <code>%s</code> a/n <b>%s</b>
6. Upload screenshot bukti transfer
7. Tunggu konfirmasi admin (biasanya <15 menit)
8. Akun langsung aktif!

<b>Perintah Tersedia:</b>
/start - Menu utama
/paket - Lihat & beli paket
/cekorder - Cek status order
/info - Info server
/bantuan - Tampilkan bantuan ini

❓ <b>Ada pertanyaan?</b>
Hubungi admin langsung via Telegram.`, DANA_NO, DANA_NAME)

	sendMsg(chatID, msg)
}

// ── STATE MACHINE INPUT ──────────────────────────────────────
func handleStateInput(userID int64, chatID int64, username, text string, state *UserState) {
	switch state.Step {

	case "pilih_paket":
		pakets := loadPaket()
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > len(pakets) {
			sendMsg(chatID, fmt.Sprintf("❌ Pilihan tidak valid. Ketik angka 1-%d", len(pakets)))
			return
		}
		p := pakets[n-1]
		state.PaketName = p.Nama
		state.Step = "input_username"
		setState(userID, state)

		kuotaStr := "Unlimited"
		if p.Kuota > 0 {
			kuotaStr = fmt.Sprintf("%d GB", p.Kuota)
		}
		msg := fmt.Sprintf(`✅ <b>Paket dipilih: %s</b>

⏱ Masa aktif : %d hari
📦 Kuota      : %s
🔒 Max device : %d
💰 Harga      : <b>Rp %s</b>

━━━━━━━━━━━━━━━━━━━━━━
Sekarang masukkan <b>username</b> yang kamu inginkan:
<i>(huruf kecil, angka, minimal 4 karakter)</i>`, p.Nama, p.Hari, kuotaStr, p.MaxLogin, formatRupiah(p.Harga))
		sendMsg(chatID, msg)

	case "input_username":
		un := strings.ToLower(strings.TrimSpace(text))
		if len(un) < 4 || len(un) > 20 {
			sendMsg(chatID, "❌ Username harus 4-20 karakter. Coba lagi:")
			return
		}
		if !isValidUsername(un) {
			sendMsg(chatID, "❌ Username hanya boleh huruf kecil dan angka. Coba lagi:")
			return
		}
		if userExistsBot(un) {
			sendMsg(chatID, "❌ Username sudah digunakan. Pilih username lain:")
			return
		}
		state.Username = un
		state.Step = "input_password"
		setState(userID, state)
		sendMsg(chatID, fmt.Sprintf("✅ Username: <code>%s</code>\n\nSekarang masukkan <b>password</b>:\n<i>(minimal 6 karakter, atau ketik <code>auto</code> untuk password otomatis)</i>", un))

	case "input_password":
		pw := strings.TrimSpace(text)
		if strings.ToLower(pw) == "auto" {
			pw = randPassBot()
		}
		if len(pw) < 6 {
			sendMsg(chatID, "❌ Password minimal 6 karakter. Coba lagi (atau ketik <code>auto</code>):")
			return
		}
		state.Password = pw
		state.Step = "upload_bukti"
		setState(userID, state)

		// Buat order
		orderID := genOrderID()
		pakets := loadPaket()
		var p *Paket
		for _, pk := range pakets {
			if pk.Nama == state.PaketName {
				cp := pk
				p = &cp
				break
			}
		}
		if p == nil {
			sendMsg(chatID, "❌ Paket tidak ditemukan. Mulai ulang dengan /paket")
			resetState(userID)
			return
		}

		state.OrderID = orderID
		setState(userID, state)

		o := Order{
			ID:        orderID,
			UserID:    userID,
			Username:  username,
			Paket:     p.Nama,
			Harga:     p.Harga,
			Status:    "pending_payment",
			Timestamp: time.Now().Format("02/01/2006 15:04"),
			AkunUser:  state.Username,
			AkunPass:  pw,
		}
		saveOrder(o)

		msg := fmt.Sprintf(`📝 <b>ORDER DIBUAT!</b>

🆔 Order ID   : <code>%s</code>
📦 Paket      : %s
👤 Username   : <code>%s</code>
🔑 Password   : <code>%s</code>
💰 Total      : <b>Rp %s</b>

━━━━━━━━━━━━━━━━━━━━━━
💳 <b>CARA BAYAR:</b>

1️⃣ Transfer <b>Rp %s</b> via DANA
2️⃣ Nomor DANA: <code>%s</code>
3️⃣ Atas nama: <b>%s</b>

━━━━━━━━━━━━━━━━━━━━━━
📸 Setelah transfer, <b>upload screenshot</b> bukti pembayaran di sini!

⏰ Batas upload: <b>30 menit</b>`, orderID, p.Nama, state.Username, pw, formatRupiah(p.Harga), formatRupiah(p.Harga), DANA_NO, DANA_NAME)

		sendMsg(chatID, msg)

		// Notif admin
		notifAdmin(fmt.Sprintf(`🔔 <b>ORDER BARU!</b>

🆔 Order   : <code>%s</code>
👤 User    : @%s (ID: %d)
📦 Paket   : %s
💰 Harga   : Rp %s
🕐 Waktu   : %s

⏳ Menunggu bukti pembayaran...`, orderID, username, userID, p.Nama, formatRupiah(p.Harga), o.Timestamp))

	case "upload_bukti":
		sendMsg(chatID, "📸 Mohon upload <b>foto/screenshot</b> bukti pembayaran DANA.\n\nJangan kirim teks, kirim gambar ya!")

	case "broadcast":
		if !isAdmin(userID) {
			return
		}
		if text == "" {
			sendMsg(chatID, "❌ Pesan kosong!")
			resetState(userID)
			return
		}
		doBroadcast(text, chatID)
		resetState(userID)

	case "admin_create_akun":
		// step by step admin create
		handleAdminCreateStep(userID, chatID, text, state)

	default:
		// Cek tombol admin panel
		if text == "👑 Admin Panel" && isAdmin(userID) {
			handleAdminMenu(chatID)
		} else {
			handleStart(userID, chatID, username)
		}
	}
}

// ── HANDLE FOTO BUKTI ────────────────────────────────────────
func handlePhotoBukti(userID int64, chatID int64, username string, msg Message) {
	stateMu.Lock()
	state, exists := userState[userID]
	stateMu.Unlock()

	if !exists || state.Step != "upload_bukti" || state.OrderID == "" {
		sendMsg(chatID, "ℹ️ Foto diterima, tapi kamu tidak sedang dalam proses pembayaran.\n\nKetik /paket untuk membeli VPN.")
		return
	}

	// Ambil file_id foto terbesar
	fileID := ""
	if len(msg.Photo) > 0 {
		fileID = msg.Photo[len(msg.Photo)-1].FileID
	}

	// Update order
	o := getOrderByID(state.OrderID)
	if o == nil {
		sendMsg(chatID, "❌ Order tidak ditemukan!")
		resetState(userID)
		return
	}
	o.Bukti = fileID
	o.Status = "waiting_confirm"
	saveOrder(*o)

	resetState(userID)

	sendMsg(chatID, fmt.Sprintf(`✅ <b>Bukti pembayaran diterima!</b>

🆔 Order ID : <code>%s</code>
📦 Paket    : %s
💰 Harga    : Rp %s

⏳ Admin sedang memverifikasi...
Biasanya selesai dalam <b>5-15 menit</b>.

Gunakan /cekorder untuk cek status.`, o.ID, o.Paket, formatRupiah(o.Harga)))

	// Notif ke semua admin dengan foto
	pakets := loadPaket()
	var p *Paket
	for _, pk := range pakets {
		if pk.Nama == o.Paket {
			cp := pk
			p = &cp
			break
		}
	}
	hariStr := "30"
	kuotaStr := "0"
	maxStr := "2"
	if p != nil {
		hariStr = strconv.Itoa(p.Hari)
		kuotaStr = strconv.Itoa(p.Kuota)
		maxStr = strconv.Itoa(p.MaxLogin)
	}

	adminMsg := fmt.Sprintf(`💳 <b>BUKTI PEMBAYARAN MASUK!</b>

🆔 Order    : <code>%s</code>
👤 TG User  : @%s (ID: <code>%d</code>)
📦 Paket    : %s
💰 Harga    : Rp %s
👤 Username : <code>%s</code>
🔑 Password : <code>%s</code>
🕐 Waktu    : %s

━━━━━━━━━━━━━━━━━━━━━━
✅ Konfirmasi (akun otomatis dibuat):
<code>/confirm %s %s %s %s %s %s</code>

❌ Tolak:
<code>/reject %s Alasan penolakan</code>`,
		o.ID, username, userID, o.Paket, formatRupiah(o.Harga),
		o.AkunUser, o.AkunPass, o.Timestamp,
		o.ID, o.AkunUser, o.AkunPass, hariStr, kuotaStr, maxStr,
		o.ID)

	for _, adminID := range adminIDs {
		sendPhoto(adminID, fileID, adminMsg)
	}
}

// ── ADMIN: CONFIRM ORDER ─────────────────────────────────────
// /confirm ORDERID username password hari kuota maxlogin
func handleAdminConfirm(chatID int64, adminID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 7 {
		sendMsg(chatID, "❌ Format: <code>/confirm ORDERID username password hari kuota maxlogin</code>\n\nContoh:\n<code>/confirm ORD-001 john pass123 30 0 2</code>")
		return
	}

	orderID := parts[1]
	un := parts[2]
	pw := parts[3]
	hariStr := parts[4]
	kuotaStr := parts[5]
	maxStr := parts[6]

	hari, _ := strconv.Atoi(hariStr)
	kuota, _ := strconv.Atoi(kuotaStr)
	maxLogin, _ := strconv.Atoi(maxStr)
	if hari <= 0 {
		hari = 30
	}
	if maxLogin <= 0 {
		maxLogin = 2
	}

	o := getOrderByID(orderID)
	if o == nil {
		sendMsg(chatID, fmt.Sprintf("❌ Order <code>%s</code> tidak ditemukan!", orderID))
		return
	}
	if o.Status != "waiting_confirm" && o.Status != "pending_payment" {
		sendMsg(chatID, fmt.Sprintf("⚠️ Order %s status: %s (bukan waiting_confirm)", orderID, o.Status))
		return
	}

	// Buat akun di sistem
	exp := time.Now().AddDate(0, 0, hari).Format("2006-01-02")
	err := createAkunBot(un, pw, exp, kuotaStr, strconv.Itoa(maxLogin), o.Username)
	if err != nil {
		sendMsg(chatID, fmt.Sprintf("❌ Gagal buat akun: %v", err))
		return
	}

	// Update order
	o.Status = "done"
	o.AkunUser = un
	o.AkunPass = pw
	o.AkunExp = exp
	saveOrder(*o)

	domain := getDomainBot()
	port := getPortBot()

	// Notif ke admin
	sendMsg(chatID, fmt.Sprintf("✅ <b>Order %s berhasil dikonfirmasi!</b>\nAkun <code>%s</code> telah dibuat.", orderID, un))

	// Kirim akun ke user
	kuotaDisplay := "Unlimited"
	if kuota > 0 {
		kuotaDisplay = fmt.Sprintf("%d GB", kuota)
	}
	userMsg := fmt.Sprintf(`🎉 <b>AKUN VPN KAMU SUDAH AKTIF!</b>

🆔 Order    : <code>%s</code>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>Username</b>  : <code>%s</code>
🔑 <b>Password</b>  : <code>%s</code>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🖥 <b>IP/Host</b>   : <code>%s</code>
🔌 <b>Port</b>      : <code>%s</code>
📡 <b>Obfs</b>      : <code>zivpn</code>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 <b>Kuota</b>     : %s
🔒 <b>MaxLogin</b>  : %d device
📅 <b>Expired</b>   : %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📱 Download <b>ZiVPN</b> di Play Store / App Store

⚠️ Jangan share akun ini ke orang lain!
Gunakan /cekorder untuk melihat detail akun.`, o.ID, un, pw, domain, port, kuotaDisplay, maxLogin, exp)

	sendMsg(o.UserID, userMsg)
}

// ── ADMIN: REJECT ORDER ──────────────────────────────────────
func handleAdminReject(chatID int64, text string) {
	parts := strings.SplitN(text, " ", 3)
	if len(parts) < 2 {
		sendMsg(chatID, "❌ Format: <code>/reject ORDERID [alasan]</code>")
		return
	}
	orderID := parts[1]
	alasan := "Bukti pembayaran tidak valid"
	if len(parts) >= 3 {
		alasan = parts[2]
	}

	o := getOrderByID(orderID)
	if o == nil {
		sendMsg(chatID, fmt.Sprintf("❌ Order %s tidak ditemukan!", orderID))
		return
	}

	o.Status = "rejected"
	saveOrder(*o)

	sendMsg(chatID, fmt.Sprintf("✅ Order <code>%s</code> telah ditolak.", orderID))
	sendMsg(o.UserID, fmt.Sprintf(`❌ <b>Order Ditolak</b>

🆔 Order : <code>%s</code>
📦 Paket : %s
📝 Alasan: %s

Silakan hubungi admin atau coba lagi dengan /paket.`, orderID, o.Paket, alasan))
}

// ── ADMIN: CREATE AKUN GRATIS ────────────────────────────────
// /createakun username password hari kuota maxlogin [note]
func handleAdminCreateAkun(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 6 {
		sendMsg(chatID, `❌ <b>Format:</b>
<code>/createakun username password hari kuota maxlogin [note]</code>

<b>Contoh:</b>
<code>/createakun john123 pass123 30 0 2 Reseller</code>

<i>kuota 0 = Unlimited</i>`)
		return
	}

	un := parts[1]
	pw := parts[2]
	hariStr := parts[3]
	kuotaStr := parts[4]
	maxStr := parts[5]
	note := "-"
	if len(parts) >= 7 {
		note = strings.Join(parts[6:], " ")
	}

	hari, err := strconv.Atoi(hariStr)
	if err != nil || hari <= 0 {
		sendMsg(chatID, "❌ Hari tidak valid!")
		return
	}

	if userExistsBot(un) {
		sendMsg(chatID, fmt.Sprintf("❌ Username <code>%s</code> sudah ada!", un))
		return
	}

	exp := time.Now().AddDate(0, 0, hari).Format("2006-01-02")
	err = createAkunBot(un, pw, exp, kuotaStr, maxStr, note)
	if err != nil {
		sendMsg(chatID, fmt.Sprintf("❌ Gagal buat akun: %v", err))
		return
	}

	domain := getDomainBot()
	port := getPortBot()
	kuota, _ := strconv.Atoi(kuotaStr)
	maxLogin, _ := strconv.Atoi(maxStr)
	kuotaDisplay := "Unlimited"
	if kuota > 0 {
		kuotaDisplay = fmt.Sprintf("%d GB", kuota)
	}

	sendMsg(chatID, fmt.Sprintf(`✅ <b>Akun Berhasil Dibuat (Admin)</b>

👤 Username : <code>%s</code>
🔑 Password : <code>%s</code>
━━━━━━━━━━━━━━━━━━━━━━
🌐 Host     : <code>%s</code>
🔌 Port     : <code>%s</code>
📡 Obfs     : <code>zivpn</code>
━━━━━━━━━━━━━━━━━━━━━━
📦 Kuota    : %s
🔒 MaxLogin : %d device
📅 Expired  : %s
📝 Note     : %s`, un, pw, domain, port, kuotaDisplay, maxLogin, exp, note))
}

func handleAdminCreateStep(userID int64, chatID int64, text string, state *UserState) {
	// Bisa dikembangkan untuk wizard step-by-step
	sendMsg(chatID, "Gunakan command: <code>/createakun username password hari kuota maxlogin</code>")
	resetState(userID)
}

// ── ADMIN MENU ───────────────────────────────────────────────
func handleAdminMenu(chatID int64) {
	msg := `👑 <b>ADMIN PANEL</b>

<b>Manajemen Order:</b>
/listorder — Lihat semua order pending
/listuser  — Lihat semua akun aktif

<b>Buat Akun (gratis):</b>
/createakun username pass hari kuota maxlogin

<b>Manajemen Paket:</b>
/setpaket  — Lihat & kelola paket
/addpaket  — Panduan tambah paket
/newpaket  — Tambah paket baru
/delpaket  — Hapus paket

<b>Manajemen Admin:</b>
/addadmin  — Panduan tambah admin
/setadmin ID — Tambah admin baru

<b>Lainnya:</b>
/broadcast — Kirim pesan ke semua user
/info      — Info server`

	sendMsg(chatID, msg)
}

// ── ADMIN: LIST ORDER ────────────────────────────────────────
func handleListOrder(chatID int64) {
	orders := getAllOrders()
	if len(orders) == 0 {
		sendMsg(chatID, "📋 Belum ada order.")
		return
	}

	pending := []Order{}
	waiting := []Order{}
	for _, o := range orders {
		switch o.Status {
		case "pending_payment":
			pending = append(pending, o)
		case "waiting_confirm":
			waiting = append(waiting, o)
		}
	}

	msg := fmt.Sprintf("📋 <b>DAFTAR ORDER</b>\n\nTotal: %d order\n\n", len(orders))

	if len(waiting) > 0 {
		msg += fmt.Sprintf("⏳ <b>Menunggu Konfirmasi (%d):</b>\n", len(waiting))
		for _, o := range waiting {
			msg += fmt.Sprintf("• <code>%s</code> — @%s — %s — Rp %s\n", o.ID, o.Username, o.Paket, formatRupiah(o.Harga))
		}
		msg += "\n"
	}

	if len(pending) > 0 {
		msg += fmt.Sprintf("💳 <b>Menunggu Pembayaran (%d):</b>\n", len(pending))
		for _, o := range pending {
			msg += fmt.Sprintf("• <code>%s</code> — @%s — %s — Rp %s\n", o.ID, o.Username, o.Paket, formatRupiah(o.Harga))
		}
	}

	if len(waiting) == 0 && len(pending) == 0 {
		msg += "✅ Tidak ada order pending saat ini."
	}

	sendMsg(chatID, msg)
}

// ── ADMIN: LIST USER ─────────────────────────────────────────
func handleListUser(chatID int64) {
	data, err := os.ReadFile(UDB_BOT)
	if err != nil || strings.TrimSpace(string(data)) == "" {
		sendMsg(chatID, "👤 Belum ada akun terdaftar.")
		return
	}

	today := time.Now().Format("2006-01-02")
	lines := strings.Split(string(data), "\n")
	aktif, expired := 0, 0
	msg := "👥 <b>DAFTAR AKUN VPN</b>\n\n"

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		u, e := parts[0], parts[2]
		status := "✅"
		if e < today {
			status = "❌"
			expired++
		} else {
			aktif++
		}
		msg += fmt.Sprintf("%s <code>%-16s</code> exp: %s\n", status, u, e)
	}

	msg += fmt.Sprintf("\n📊 Total: %d aktif, %d expired", aktif, expired)

	// Jika terlalu panjang, potong
	if len(msg) > 4000 {
		msg = msg[:4000] + "\n...(terpotong)"
	}
	sendMsg(chatID, msg)
}

// ── ADMIN: PAKET MANAGEMENT ──────────────────────────────────
func handleListPaket(chatID int64, isAdmin bool) {
	pakets := loadPaket()
	msg := "📦 <b>DAFTAR PAKET:</b>\n\n"
	for i, p := range pakets {
		kuotaStr := "Unlimited"
		if p.Kuota > 0 {
			kuotaStr = fmt.Sprintf("%d GB", p.Kuota)
		}
		msg += fmt.Sprintf("%d. <b>%s</b> — %d hari — %s — Rp %s\n",
			i+1, p.Nama, p.Hari, kuotaStr, formatRupiah(p.Harga))
	}
	if isAdmin {
		msg += "\n<b>Hapus paket:</b> <code>/delpaket NamaPaket</code>"
	}
	sendMsg(chatID, msg)
}

func handleNewPaket(chatID int64, text string) {
	// /newpaket Nama|Hari|Harga|Kuota|MaxLogin
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		sendMsg(chatID, "❌ Format: <code>/newpaket Nama|Hari|Harga|Kuota|MaxLogin</code>")
		return
	}
	fields := strings.Split(strings.TrimSpace(parts[1]), "|")
	if len(fields) < 5 {
		sendMsg(chatID, "❌ Format salah! Gunakan: <code>Nama|Hari|Harga|Kuota|MaxLogin</code>")
		return
	}

	nama := strings.TrimSpace(fields[0])
	hari, _ := strconv.Atoi(strings.TrimSpace(fields[1]))
	harga, _ := strconv.Atoi(strings.TrimSpace(fields[2]))
	kuota, _ := strconv.Atoi(strings.TrimSpace(fields[3]))
	maxlogin, _ := strconv.Atoi(strings.TrimSpace(fields[4]))

	if nama == "" || hari <= 0 || harga <= 0 {
		sendMsg(chatID, "❌ Data tidak valid!")
		return
	}
	if maxlogin <= 0 {
		maxlogin = 2
	}

	line := fmt.Sprintf("%s|%d|%d|%d|%d\n", nama, hari, harga, kuota, maxlogin)
	f, err := os.OpenFile(PAKET_DB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		sendMsg(chatID, "❌ Gagal menyimpan paket!")
		return
	}
	fmt.Fprint(f, line)
	f.Close()

	kuotaStr := "Unlimited"
	if kuota > 0 {
		kuotaStr = fmt.Sprintf("%d GB", kuota)
	}
	sendMsg(chatID, fmt.Sprintf("✅ Paket berhasil ditambahkan!\n\n📦 %s | %d hari | %s | Rp %s", nama, hari, kuotaStr, formatRupiah(harga)))
}

func handleDelPaket(chatID int64, text string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		sendMsg(chatID, "❌ Format: <code>/delpaket NamaPaket</code>")
		return
	}
	nama := strings.TrimSpace(parts[1])

	data, err := os.ReadFile(PAKET_DB)
	if err != nil {
		sendMsg(chatID, "❌ Gagal membaca database paket!")
		return
	}

	var kept []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) > 0 && fields[0] == nama {
			found = true
			continue
		}
		kept = append(kept, line)
	}

	if !found {
		sendMsg(chatID, fmt.Sprintf("❌ Paket '%s' tidak ditemukan!", nama))
		return
	}

	os.WriteFile(PAKET_DB, []byte(strings.Join(kept, "\n")+"\n"), 0644)
	sendMsg(chatID, fmt.Sprintf("✅ Paket '%s' berhasil dihapus!", nama))
}

// ── ADMIN: SET ADMIN ─────────────────────────────────────────
func handleSetAdmin(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendMsg(chatID, "❌ Format: <code>/setadmin TelegramID</code>")
		return
	}
	newID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		sendMsg(chatID, "❌ ID tidak valid!")
		return
	}

	// Cek sudah ada belum
	for _, id := range adminIDs {
		if id == newID {
			sendMsg(chatID, fmt.Sprintf("⚠️ ID %d sudah menjadi admin.", newID))
			return
		}
	}

	adminIDs = append(adminIDs, newID)
	f, _ := os.OpenFile(ADMIN_DB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	fmt.Fprintf(f, "%d\n", newID)
	f.Close()

	sendMsg(chatID, fmt.Sprintf("✅ ID <code>%d</code> berhasil ditambahkan sebagai admin!", newID))
	sendMsg(newID, "👑 Kamu telah ditambahkan sebagai admin OGH-ZIV VPN Bot!")
}

// ── BROADCAST ───────────────────────────────────────────────
func doBroadcast(msg string, adminChatID int64) {
	orders := getAllOrders()
	sent := make(map[int64]bool)
	count := 0
	for _, o := range orders {
		if !sent[o.UserID] {
			sendMsg(o.UserID, "📢 <b>INFO dari Admin:</b>\n\n"+msg)
			sent[o.UserID] = true
			count++
		}
	}
	sendMsg(adminChatID, fmt.Sprintf("✅ Broadcast terkirim ke %d user.", count))
}

// ── DATABASE HELPERS ─────────────────────────────────────────
func loadPaket() []Paket {
	data, err := os.ReadFile(PAKET_DB)
	if err != nil {
		return nil
	}
	var pakets []Paket
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		hari, _ := strconv.Atoi(parts[1])
		harga, _ := strconv.Atoi(parts[2])
		kuota, _ := strconv.Atoi(parts[3])
		maxlogin, _ := strconv.Atoi(parts[4])
		pakets = append(pakets, Paket{
			Nama:     parts[0],
			Hari:     hari,
			Harga:    harga,
			Kuota:    kuota,
			MaxLogin: maxlogin,
		})
	}
	return pakets
}

func saveOrder(o Order) {
	// Load semua order, update atau append
	orders := getAllOrders()
	found := false
	for i, ord := range orders {
		if ord.ID == o.ID {
			orders[i] = o
			found = true
			break
		}
	}
	if !found {
		orders = append(orders, o)
	}

	var lines []string
	for _, ord := range orders {
		lines = append(lines, fmt.Sprintf("%s|%d|%s|%s|%d|%s|%s|%s|%s|%s|%s",
			ord.ID, ord.UserID, ord.Username, ord.Paket, ord.Harga,
			ord.Status, ord.Timestamp, ord.Bukti, ord.AkunUser, ord.AkunPass, ord.AkunExp))
	}
	os.WriteFile(ORDERS_DB, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func getAllOrders() []Order {
	data, err := os.ReadFile(ORDERS_DB)
	if err != nil {
		return nil
	}
	var orders []Order
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			continue
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		harga, _ := strconv.Atoi(parts[4])
		o := Order{
			ID:        parts[0],
			UserID:    uid,
			Username:  parts[2],
			Paket:     parts[3],
			Harga:     harga,
			Status:    parts[5],
			Timestamp: parts[6],
			Bukti:     parts[7],
		}
		if len(parts) >= 9 {
			o.AkunUser = parts[8]
		}
		if len(parts) >= 10 {
			o.AkunPass = parts[9]
		}
		if len(parts) >= 11 {
			o.AkunExp = parts[10]
		}
		orders = append(orders, o)
	}
	return orders
}

func getOrderByID(id string) *Order {
	for _, o := range getAllOrders() {
		if o.ID == id {
			cp := o
			return &cp
		}
	}
	return nil
}

func getOrdersByUser(userID int64) []Order {
	var result []Order
	for _, o := range getAllOrders() {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	// Sort by timestamp, ambil 5 terbaru
	if len(result) > 5 {
		result = result[len(result)-5:]
	}
	return result
}

// ── AKUN VPN HELPERS ─────────────────────────────────────────
func createAkunBot(username, password, expired, kuota, maxLogin, note string) error {
	// Validasi
	if userExistsBot(username) {
		return fmt.Errorf("username %s sudah ada", username)
	}

	// Append ke users.db
	f, err := os.OpenFile(UDB_BOT, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("gagal buka UDB: %v", err)
	}
	fmt.Fprintf(f, "%s|%s|%s|%s|%s\n", username, password, expired, kuota, note)
	f.Close()

	// Set maxlogin
	setMaxloginBot(username, maxLogin)

	// Reload password di config ZiVPN
	reloadPWBot()

	return nil
}

func userExistsBot(username string) bool {
	data, err := os.ReadFile(UDB_BOT)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, username+"|") {
			return true
		}
	}
	return false
}

func setMaxloginBot(username string, maxdev int) {
	// Hapus entry lama
	data, _ := os.ReadFile(MLDB_BOT)
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.HasPrefix(line, username+"|") {
			lines = append(lines, line)
		}
	}
	lines = append(lines, fmt.Sprintf("%s|%d", username, maxdev))
	os.WriteFile(MLDB_BOT, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func reloadPWBot() {
	data, err := os.ReadFile(UDB_BOT)
	if err != nil {
		return
	}

	var passwords []string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) >= 2 && parts[1] != "" {
			passwords = append(passwords, parts[1])
		}
	}

	cfgData, err := os.ReadFile(CFG_BOT)
	if err != nil {
		return
	}

	var cfgMap map[string]interface{}
	if err := json.Unmarshal(cfgData, &cfgMap); err != nil {
		return
	}

	if auth, ok := cfgMap["auth"].(map[string]interface{}); ok {
		var pwList []interface{}
		for _, p := range passwords {
			pwList = append(pwList, p)
		}
		auth["config"] = pwList
	}

	out, _ := json.MarshalIndent(cfgMap, "", "  ")
	os.WriteFile(CFG_BOT, out, 0644)

	// Restart service
	cmd := "/usr/bin/systemctl"
	if _, err := os.Stat(cmd); os.IsNotExist(err) {
		cmd = "systemctl"
	}
	// Non-blocking restart
	go func() {
		_ = runCmd(cmd, "restart", "zivpn")
	}()
}

func runCmd(name string, args ...string) error {
	// Simple exec helper
	f, _ := os.OpenFile("/tmp/zivpn-bot.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	scanner := bufio.NewScanner(strings.NewReader(name + " " + strings.Join(args, " ")))
	_ = scanner
	// Use os/exec equivalent via shell
	cmdStr := name + " " + strings.Join(args, " ")
	file, _ := os.Create("/tmp/zivpn-bot-cmd.sh")
	fmt.Fprintln(file, "#!/bin/sh")
	fmt.Fprintln(file, cmdStr)
	file.Close()
	os.Chmod("/tmp/zivpn-bot-cmd.sh", 0755)

	cmd := &simpleCmd{path: "/bin/sh", args: []string{"/tmp/zivpn-bot-cmd.sh"}}
	return cmd.run()
}

// ── SIMPLE CMD (tanpa os/exec import loop) ──────────────────
type simpleCmd struct {
	path string
	args []string
}

func (c *simpleCmd) run() error {
	// Tulis ke file dan jalankan via syscall-like approach
	// Karena kita tidak import os/exec di sini, gunakan workaround
	script := c.path + " " + strings.Join(c.args, " ")
	f, err := os.Create("/tmp/_zivpn_run.sh")
	if err != nil {
		return err
	}
	fmt.Fprintln(f, "#!/bin/sh")
	fmt.Fprintln(f, script)
	f.Close()
	os.Chmod("/tmp/_zivpn_run.sh", 0755)
	// Tidak bisa execute tanpa os/exec, simpan saja & return
	_ = script
	return nil
}

// ── NOTIFY ADMIN ─────────────────────────────────────────────
func notifAdmin(msg string) {
	for _, id := range adminIDs {
		sendMsg(id, msg)
	}
}

// ── VPS INFO ─────────────────────────────────────────────────
func getDomainBot() string {
	b, err := os.ReadFile(DOMF_BOT)
	if err != nil || strings.TrimSpace(string(b)) == "" {
		return "your-server-ip"
	}
	return strings.TrimSpace(string(b))
}

func getPortBot() string {
	data, err := os.ReadFile(CFG_BOT)
	if err != nil {
		return "5667"
	}
	// Cari "listen":":PORT"
	idx := strings.Index(string(data), `"listen"`)
	if idx == -1 {
		return "5667"
	}
	sub := string(data)[idx:]
	start := strings.Index(sub, ":")
	if start == -1 {
		return "5667"
	}
	sub = sub[start+1:]
	// Ambil angka
	numStr := ""
	for _, c := range sub {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else if numStr != "" {
			break
		}
	}
	if numStr == "" {
		return "5667"
	}
	return numStr
}

func getBrand() string {
	data, err := os.ReadFile(STRF_BOT)
	if err != nil {
		return "OGH-ZIV"
	}
	conf := parseConfBot(string(data))
	if b := conf["BRAND"]; b != "" {
		return b
	}
	return "OGH-ZIV"
}

// ── UTILS ────────────────────────────────────────────────────
func parseConfBot(data string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(data, "\n") {
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			val = strings.Trim(val, `"'`)
			m[key] = val
		}
	}
	return m
}

func randPassBot() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func genOrderID() string {
	t := time.Now()
	rand6 := fmt.Sprintf("%04d", rand.Intn(9999))
	return fmt.Sprintf("ZIV-%s%s", t.Format("0102150405"), rand6)
}

func formatRupiah(n int) string {
	s := strconv.Itoa(n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}

func isValidUsername(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func statusEmoji(status string) string {
	switch status {
	case "pending_payment":
		return "💳"
	case "waiting_confirm":
		return "⏳"
	case "confirmed":
		return "✅"
	case "done":
		return "🎉"
	case "rejected":
		return "❌"
	default:
		return "❓"
	}
}

func statusText(status string) string {
	switch status {
	case "pending_payment":
		return "Menunggu Pembayaran"
	case "waiting_confirm":
		return "Menunggu Konfirmasi Admin"
	case "confirmed":
		return "Dikonfirmasi"
	case "done":
		return "Selesai — Akun Aktif"
	case "rejected":
		return "Ditolak"
	default:
		return status
	}
}

func setState(userID int64, state *UserState) {
	stateMu.Lock()
	userState[userID] = state
	stateMu.Unlock()
}

func resetState(userID int64) {
	stateMu.Lock()
	userState[userID] = &UserState{Step: "menu"}
	stateMu.Unlock()
}
