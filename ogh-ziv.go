package main

// ============================================================
//   OGH-ZIV Premium Panel — Go Edition
//   Creator : OGH-ZIV Team
//   Ketik   : menu  untuk membuka panel
//   Support : Debian (all version) & Ubuntu (all version)
// ============================================================

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ── KONSTANTA & PATH ────────────────────────────────────────
const (
	DIR        = "/etc/zivpn"
	CFG        = "/etc/zivpn/config.json"
	BIN        = "/usr/local/bin/zivpn-bin"
	SVC        = "/etc/systemd/system/zivpn.service"
	LOG        = "/etc/zivpn/zivpn.log"
	UDB        = "/etc/zivpn/users.db"
	DOMF       = "/etc/zivpn/domain.conf"
	BOTF       = "/etc/zivpn/bot.conf"
	STRF       = "/etc/zivpn/store.conf"
	THEMEF     = "/etc/zivpn/theme.conf"
	MLDB       = "/etc/zivpn/maxlogin.db"
	BINARY_URL = "https://github.com/fauzanihanipah/ziv-udp/releases/download/udp-zivpn/udp-zivpn-linux-amd64"
	CONFIG_URL = "https://raw.githubusercontent.com/fauzanihanipah/ziv-udp/main/config.json"
)

// ── WARNA TEMA ─────────────────────────────────────────────
var (
	A1, A2, A3, A4, AL, AT string
	NC, BLD, DIM, IT       string
	W, LG, LR, LC, Y       string
	THEME_NAME              string
)

func loadTheme() {
	theme := "1"
	if b, err := os.ReadFile(THEMEF); err == nil {
		theme = strings.TrimSpace(string(b))
	}
	NC = "\033[0m"
	BLD = "\033[1m"
	DIM = "\033[2m"
	IT = "\033[3m"
	W = "\033[1;37m"
	LG = "\033[1;32m"
	LR = "\033[1;31m"
	LC = "\033[1;36m"
	Y = "\033[1;33m"

	switch theme {
	case "2":
		A1 = "\033[38;5;51m"; A2 = "\033[1;36m"; A3 = "\033[0;36m"; A4 = "\033[1;33m"
		AL = "\033[38;5;87m"; AT = "\033[1;37m"; THEME_NAME = "CYAN"
	case "3":
		A1 = "\033[38;5;46m"; A2 = "\033[1;32m"; A3 = "\033[0;32m"; A4 = "\033[1;33m"
		AL = "\033[38;5;82m"; AT = "\033[1;37m"; THEME_NAME = "GREEN"
	case "4":
		A1 = "\033[38;5;220m"; A2 = "\033[1;33m"; A3 = "\033[38;5;214m"; A4 = "\033[0;33m"
		AL = "\033[38;5;226m"; AT = "\033[1;37m"; THEME_NAME = "GOLD"
	case "5":
		A1 = "\033[38;5;196m"; A2 = "\033[1;31m"; A3 = "\033[0;31m"; A4 = "\033[1;33m"
		AL = "\033[38;5;203m"; AT = "\033[1;37m"; THEME_NAME = "RED"
	case "6":
		A1 = "\033[38;5;213m"; A2 = "\033[1;35m"; A3 = "\033[0;35m"; A4 = "\033[1;33m"
		AL = "\033[38;5;219m"; AT = "\033[1;37m"; THEME_NAME = "PINK"
	case "7":
		A1 = "\033[1;37m"; A2 = "\033[1;37m"; A3 = "\033[38;5;51m"; A4 = "\033[1;33m"
		AL = "\033[38;5;196m"; AT = "\033[1;37m"; THEME_NAME = "RAINBOW"
	default:
		A1 = "\033[38;5;135m"; A2 = "\033[1;35m"; A3 = "\033[38;5;141m"; A4 = "\033[1;33m"
		AL = "\033[38;5;141m"; AT = "\033[38;5;231m"; THEME_NAME = "VIOLET"
	}
}

// ── CEK OS ─────────────────────────────────────────────────
func checkOS() {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		fmt.Println("\n\033[1;31m✘ OS tidak dikenali! Script ini hanya untuk Debian & Ubuntu.\033[0m\n")
		os.Exit(1)
	}
	content := strings.ToLower(string(data))
	if !strings.Contains(content, "debian") && !strings.Contains(content, "ubuntu") {
		prettyName := ""
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				prettyName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
			}
		}
		fmt.Println()
		fmt.Println("\033[1;31m  ╔══════════════════════════════════════════════════════╗\033[0m")
		fmt.Println("\033[1;31m  ║   ✘  OS TIDAK DIDUKUNG!                              ║\033[0m")
		fmt.Println("\033[1;31m  ╠══════════════════════════════════════════════════════╣\033[0m")
		fmt.Printf("\033[1;31m  ║\033[0m  OS kamu : \033[1;33m%s\033[0m\n", prettyName)
		fmt.Println("\033[1;31m  ╠══════════════════════════════════════════════════════╣\033[0m")
		fmt.Println("\033[1;31m  ║\033[0m  Script ini hanya mendukung:                        \033[1;31m║\033[0m")
		fmt.Println("\033[1;31m  ║\033[0m  \033[1;32m✔\033[0m  Debian (semua versi)                          \033[1;31m║\033[0m")
		fmt.Println("\033[1;31m  ║\033[0m  \033[1;32m✔\033[0m  Ubuntu (semua versi)                          \033[1;31m║\033[0m")
		fmt.Println("\033[1;31m  ╚══════════════════════════════════════════════════════╝\033[0m")
		fmt.Println()
		os.Exit(1)
	}
}

func checkRoot() {
	if os.Geteuid() != 0 {
		fmt.Printf("\n\033[1;31m✘ Jalankan sebagai root!\033[0m\n\n")
		os.Exit(1)
	}
}

// ── UTILS ──────────────────────────────────────────────────
func ok(msg string)   { fmt.Printf("  %s✔%s  %s\n", A2, NC, msg) }
func inf(msg string)  { fmt.Printf("  %s➜%s  %s\n", A3, NC, msg) }
func warn(msg string) { fmt.Printf("  %s⚠%s  %s\n", A4, NC, msg) }
func errMsg(msg string) { fmt.Printf("  \033[1;31m✘%s  %s\n", NC, msg) }

func pause() {
	fmt.Println()
	fmt.Printf("  %s╰─ [ Enter ] kembali ke menu...%s", DIM, NC)
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func getIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ifconfig.me")
	if err == nil {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		ip := strings.TrimSpace(string(b))
		if ip != "" {
			return ip
		}
	}
	out, _ := exec.Command("hostname", "-I").Output()
	parts := strings.Fields(string(out))
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func getPort() string {
	data, err := os.ReadFile(CFG)
	if err != nil {
		return "5667"
	}
	re := regexp.MustCompile(`"listen"\s*:\s*":(\d+)"`)
	m := re.FindStringSubmatch(string(data))
	if len(m) > 1 {
		return m[1]
	}
	return "5667"
}

func getDomain() string {
	b, err := os.ReadFile(DOMF)
	if err != nil {
		return getIP()
	}
	d := strings.TrimSpace(string(b))
	if d == "" {
		return getIP()
	}
	return d
}

func isUp() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "zivpn")
	return cmd.Run() == nil
}

func totalUser() int {
	data, err := os.ReadFile(UDB)
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	count := 0
	for _, l := range lines {
		if l != "" {
			count++
		}
	}
	return count
}

func expCount() int {
	data, err := os.ReadFile(UDB)
	if err != nil {
		return 0
	}
	today := time.Now().Format("2006-01-02")
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) >= 3 && parts[2] < today {
			count++
		}
	}
	return count
}

func randPass() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ── MAXLOGIN HELPERS ───────────────────────────────────────
func getMaxlogin(username string) string {
	data, err := os.ReadFile(MLDB)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, username+"|") {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func setMaxlogin(username, maxdev string) {
	delMaxlogin(username)
	f, err := os.OpenFile(MLDB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s|%s\n", username, maxdev)
}

func delMaxlogin(username string) {
	data, err := os.ReadFile(MLDB)
	if err != nil {
		return
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.HasPrefix(line, username+"|") {
			lines = append(lines, line)
		}
	}
	os.WriteFile(MLDB, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func checkMaxloginAll() {
	data, err := os.ReadFile(MLDB)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		uname, maxDevStr := parts[0], parts[1]
		maxdev, _ := strconv.Atoi(maxDevStr)

		// Cek dari log
		active := 0
		if logData, err := os.ReadFile(LOG); err == nil {
			active = strings.Count(string(logData), "user="+uname)
		}

		if active > maxdev {
			removeUserFromDB(uname)
			delMaxlogin(uname)
			reloadPW()
			tgSend(fmt.Sprintf("🚫 <b>Auto-Delete MaxLogin</b>\n👤 User: <code>%s</code>\n⚠️ Melebihi batas %d device — akun otomatis dihapus!", uname, maxdev))
		}
	}
}

func removeUserFromDB(username string) {
	data, err := os.ReadFile(UDB)
	if err != nil {
		return
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.HasPrefix(line, username+"|") {
			lines = append(lines, line)
		}
	}
	os.WriteFile(UDB, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// ── HELPERS ────────────────────────────────────────────────
func reloadPW() {
	data, err := os.ReadFile(UDB)
	if err != nil {
		return
	}
	var passwords []string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) >= 2 && parts[1] != "" {
			passwords = append(passwords, fmt.Sprintf("%q", parts[1]))
		}
	}

	cfgData, err := os.ReadFile(CFG)
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
			// Remove surrounding quotes from Sprintf("%q")
			pwList = append(pwList, strings.Trim(p, `"`))
		}
		auth["config"] = pwList
	}

	out, _ := json.MarshalIndent(cfgMap, "", "  ")
	os.WriteFile(CFG, out, 0644)
	exec.Command("systemctl", "restart", "zivpn").Run()
}

func tgSend(msg string) {
	data, err := os.ReadFile(BOTF)
	if err != nil {
		return
	}
	conf := parseConf(string(data))
	token := conf["BOT_TOKEN"]
	chatID := conf["CHAT_ID"]
	if token == "" || chatID == "" {
		return
	}
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
		url.Values{"chat_id": {chatID}, "text": {msg}, "parse_mode": {"HTML"}},
	)
}

func tgRaw(token, chatID, msg string) {
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
		url.Values{"chat_id": {chatID}, "text": {msg}, "parse_mode": {"HTML"}},
	)
}

func parseConf(data string) map[string]string {
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

// ── PANEL BORDER HELPERS ───────────────────────────────────
const BOX_LINE = "════════════════════════════════════════════════════════"
const BOX_LINED = "╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌"

func boxTop()  { fmt.Printf("  %s╔%s╗%s\n", A1, BOX_LINE, NC) }
func boxBot()  { fmt.Printf("  %s╚%s╝%s\n", A1, BOX_LINE, NC) }
func boxSep()  { fmt.Printf("  %s╠%s╣%s\n", A1, BOX_LINE, NC) }
func boxSep0() { fmt.Printf("  %s╠%s╣%s\n", A1, BOX_LINED, NC) }

// stripANSI removes ANSI escape codes to compute visible length
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[mJKHfABCDsuhlp]|\x1b[()][AB012]|\x1b`)
	return re.ReplaceAllString(s, "")
}

func dispLen(s string) int {
	clean := stripANSI(s)
	w := 0
	for _, r := range clean {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

func boxBtn(content string) {
	const BOX_W = 56
	dlen := dispLen(content)
	pad := BOX_W - dlen
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("  %s║%s%s%s%s║%s\n", A1, NC, content, strings.Repeat(" ", pad), A1, NC)
}

// ── LOGO & HEADER ──────────────────────────────────────────
func drawLogo() {
	theme := "1"
	if b, err := os.ReadFile(THEMEF); err == nil {
		theme = strings.TrimSpace(string(b))
	}
	var L1, L2, L3, L4, L5 string
	if theme == "7" {
		L1 = "\033[38;5;196m"; L2 = "\033[38;5;214m"; L3 = "\033[38;5;226m"
		L4 = "\033[38;5;82m"; L5 = "\033[38;5;51m"
	} else {
		L1 = AL; L2 = AL; L3 = AL; L4 = AL; L5 = AL
	}
	fmt.Println()
	fmt.Printf("  %s╔════════════════════════════════════════════════════════════╗%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s%s  ___    ____  _   _    ·   ____   ___  __   __ %s  %s║%s\n", A1, NC, L1, BLD, NC, A1, NC)
	fmt.Printf("  %s║%s  %s%s / _ \\  / ___|| | | |   ·  |_  /  |_ _| \\ \\ / /%s  %s║%s\n", A1, NC, L2, BLD, NC, A1, NC)
	fmt.Printf("  %s║%s  %s%s| | | || |  _ | |_| |   ·   / /    | |   \\ V / %s  %s║%s\n", A1, NC, L3, BLD, NC, A1, NC)
	fmt.Printf("  %s║%s  %s%s| |_| || |_| ||  _  |   ·  / /__   | |    \\ /  %s  %s║%s\n", A1, NC, L4, BLD, NC, A1, NC)
	fmt.Printf("  %s║%s  %s%s \\___/  \\____||_| |_|   ·  /____| |___|    V   %s  %s║%s\n", A1, NC, L5, BLD, NC, A1, NC)
	fmt.Printf("  %s║%s  %s╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱  P R E M I U M  ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱%s  %s║%s\n", A1, NC, DIM, NC, A1, NC)
	fmt.Printf("  %s╚════════════════════════════════════════════════════════════╝%s\n", A1, NC)
}

func drawVPS() {
	ip := getIP()
	port := getPort()
	domain := getDomain()

	// RAM
	ramCmd := exec.Command("bash", "-c", "free -m | awk '/^Mem/{print $3\"|\"$2}'")
	ramOut, _ := ramCmd.Output()
	ramParts := strings.Split(strings.TrimSpace(string(ramOut)), "|")
	ramU, ramT := "?", "?"
	if len(ramParts) == 2 {
		ramU, ramT = ramParts[0], ramParts[1]
	}

	// CPU
	cpuCmd := exec.Command("bash", "-c", `top -bn1 2>/dev/null | grep "Cpu(s)" | awk '{printf "%.1f",$2}'`)
	cpuOut, _ := cpuCmd.Output()
	cpu := strings.TrimSpace(string(cpuOut))
	if cpu == "" {
		cpu = "?"
	}

	// Disk
	dfCmd := exec.Command("bash", "-c", `df -h / | awk 'NR==2{print $3"|"$2}'`)
	dfOut, _ := dfCmd.Output()
	dfParts := strings.Split(strings.TrimSpace(string(dfOut)), "|")
	du, dt := "?", "?"
	if len(dfParts) == 2 {
		du, dt = dfParts[0], dfParts[1]
	}

	// OS
	osCmd := exec.Command("bash", "-c", `. /etc/os-release 2>/dev/null && echo "$NAME $VERSION_ID"`)
	osOut, _ := osCmd.Output()
	osName := strings.TrimSpace(string(osOut))
	if osName == "" {
		osName = "Linux"
	}

	hn, _ := os.Hostname()
	total := totalUser()
	expc := expCount()
	now := time.Now().Format("15:04  02/01/2006")

	svcCol, svcIc, svcTxt := LR, "🔴", "STOPPED"
	if isUp() {
		svcCol = LG
		svcIc = "🟢"
		svcTxt = "RUNNING"
	}

	botTxt := LR + "Belum setup" + NC
	if data, err := os.ReadFile(BOTF); err == nil {
		conf := parseConf(string(data))
		if botName := conf["BOT_NAME"]; botName != "" {
			botTxt = LG + "@" + botName + NC
		}
	}

	brand := "OGH-ZIV"
	if data, err := os.ReadFile(STRF); err == nil {
		conf := parseConf(string(data))
		if b := conf["BRAND"]; b != "" {
			brand = b
		}
	}

	temaDisplay := AL + THEME_NAME + NC
	if THEME_NAME == "RAINBOW" {
		temaDisplay = "\033[38;5;196mR\033[38;5;208mA\033[38;5;226mI\033[38;5;82mN\033[38;5;51mB\033[38;5;171mO\033[38;5;213mW\033[0m"
	}

	fmt.Println()
	fmt.Printf("  %s╔════════════════════════════════════════════════════════════╗%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s%s✦ INFO VPS%s  %s%43s%s  %s║%s\n", A1, NC, BLD, AL, NC, DIM, now, NC, A1, NC)
	fmt.Printf("  %s╠╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %sHost  %s: %s%-16s%s  %sOS    %s: %s%-18s%s  %s║%s\n", A1, NC, DIM, NC, W, hn, NC, DIM, NC, W, osName, NC, A1, NC)
	fmt.Printf("  %s║%s  %sIP    %s: %s%-16s%s  %sDomain%s: %s%-18s%s  %s║%s\n", A1, NC, DIM, NC, A3, ip, NC, DIM, NC, W, domain, NC, A1, NC)
	fmt.Printf("  %s║%s  %sPort  %s: %s%-16s%s  %sBrand %s: %s%-18s%s  %s║%s\n", A1, NC, DIM, NC, Y, port, NC, DIM, NC, AL, brand, NC, A1, NC)
	fmt.Printf("  %s╠╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %sCPU%s: %s%-6s%%%s  %sRAM%s: %s%s/%sMB%s  %sDisk%s: %s%s/%s%s %s║%s\n",
		A1, NC, DIM, NC, W, cpu, NC, DIM, NC, W, ramU, ramT, NC, DIM, NC, W, du, dt, NC, A1, NC)
	fmt.Printf("  %s╠╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s %s%s%-8s%s  %sAkun:%s %s%-3d%s  %sExp:%s %s%-3d%s  %sBot:%s %-20s  %sTema:%s %-10s  %s║%s\n",
		A1, NC, svcIc, svcCol, BLD, svcTxt, NC, DIM, NC, W, total, NC, DIM, NC, LR, expc, NC, DIM, NC, botTxt, DIM, NC, temaDisplay, A1, NC)
	fmt.Printf("  %s╚════════════════════════════════════════════════════════════╝%s\n", A1, NC)
	fmt.Println()
}

func showHeader() {
	exec.Command("clear").Run()
	loadTheme()
	drawLogo()
	drawVPS()
}

// ── BINGKAI AKUN ───────────────────────────────────────────
func showAkunBox(user, pass, domain, port, quota, expired, note, ipPub, maxl string) {
	brand := "OGH-ZIV"
	if data, err := os.ReadFile(STRF); err == nil {
		conf := parseConf(string(data))
		if b := conf["BRAND"]; b != "" {
			brand = b
		}
	}

	expTime, err := time.Parse("2006-01-02", expired)
	sisaStr := LR + "Expired" + NC
	if err == nil {
		sisa := int(time.Until(expTime).Hours() / 24)
		if sisa >= 0 {
			sisaStr = fmt.Sprintf("%s%d hari lagi%s", LG, sisa, NC)
		}
	}

	fmt.Println()
	fmt.Printf("  %s┌──────────────────────────────────────────────────────────┐%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s%s  ✦ %-52s%s  %s│%s\n", A1, NC, IT, AL, brand+" — AKUN VPN PREMIUM", NC, A1, NC)
	fmt.Printf("  %s├──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Username %s %s│%s  %s%s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, BLD, W, user, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Password %s %s│%s  %s%s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, BLD, A3, pass, NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s IP Publik %s%s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, LG, ipPub, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Host     %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, domain, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Port     %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, port, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Obfs     %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, "zivpn", NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Kuota    %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, LG, quota, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s MaxLogin %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, maxl+" device", NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Expired  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, expired, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Sisa     %s %s│%s  %-41s  %s│%s\n", A1, NC, DIM, NC, A1, NC, sisaStr, A1, NC)
	if note != "-" && note != "" {
		fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
		fmt.Printf("  %s│%s %s Pembeli  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, note, NC, A1, NC)
	}
	fmt.Printf("  %s├──────────────┴───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s📱 Download ZiVPN → Play Store / App Store%s              %s│%s\n", A1, NC, DIM, NC, A1, NC)
	fmt.Printf("  %s│%s  %s⚠  Jangan share akun ini ke orang lain!%s                %s│%s\n", A1, NC, DIM, NC, A1, NC)
	fmt.Printf("  %s└──────────────────────────────────────────────────────────┘%s\n", A1, NC)
	fmt.Println()
}

// ════════════════════════════════════════════════════════════
//  USER FUNCTIONS
// ════════════════════════════════════════════════════════════

func uAdd() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s➕  TAMBAH AKUN BARU%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	un := readLine(fmt.Sprintf("  %sUsername%s               : ", A3, NC))
	if un == "" {
		errMsg("Username kosong!")
		pause()
		return
	}
	if userExists(un) {
		errMsg("Username sudah ada!")
		pause()
		return
	}

	up := readLine(fmt.Sprintf("  %sPassword%s [auto]         : ", A3, NC))
	if up == "" {
		up = randPass()
	}

	udStr := readLine(fmt.Sprintf("  %sMasa aktif (hari)%s [30]  : ", A3, NC))
	ud := 30
	if udStr != "" {
		ud, _ = strconv.Atoi(udStr)
	}
	ue := time.Now().AddDate(0, 0, ud).Format("2006-01-02")

	uqStr := readLine(fmt.Sprintf("  %sKuota GB%s (0=unlimited)  : ", A3, NC))
	if uqStr == "" {
		uqStr = "0"
	}

	note := readLine(fmt.Sprintf("  %sCatatan / Nama Pembeli%s  : ", A3, NC))
	if note == "" {
		note = "-"
	}

	umlStr := readLine(fmt.Sprintf("  %sMax Login Device%s [2]    : ", A3, NC))
	uml := "2"
	if umlStr != "" {
		if n, err := strconv.Atoi(umlStr); err == nil && n >= 0 {
			uml = umlStr
		}
	}

	f, err := os.OpenFile(UDB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		errMsg("Gagal menulis database!")
		pause()
		return
	}
	fmt.Fprintf(f, "%s|%s|%s|%s|%s\n", un, up, ue, uqStr, note)
	f.Close()

	setMaxlogin(un, uml)
	reloadPW()

	domain := getDomain()
	port := getPort()
	ipPub := getIP()
	ql := "Unlimited"
	if uqStr != "0" {
		ql = uqStr + " GB"
	}

	brand := "OGH-ZIV"
	if data, err := os.ReadFile(STRF); err == nil {
		if b := parseConf(string(data))["BRAND"]; b != "" {
			brand = b
		}
	}

	tgSend(fmt.Sprintf("✅ <b>Akun Baru — %s</b>\n┌──────────────────────────\n│ 👤 <b>Username</b> : <code>%s</code>\n│ 🔑 <b>Password</b> : <code>%s</code>\n├──────────────────────────\n│ 🖥 <b>IP Publik</b> : <code>%s</code>\n│ 🌐 <b>Host</b>     : <code>%s</code>\n│ 🔌 <b>Port</b>     : <code>%s</code>\n│ 📡 <b>Obfs</b>     : <code>zivpn</code>\n├──────────────────────────\n│ 📦 <b>Kuota</b>    : %s\n│ 🔒 <b>MaxLogin</b> : %s device\n│ 📅 <b>Expired</b>  : %s\n│ 📝 <b>Pembeli</b>  : %s\n└──────────────────────────",
		brand, un, up, ipPub, domain, port, ql, uml, ue, note))

	showAkunBox(un, up, domain, port, ql, ue, note, ipPub, uml)
	pause()
}

func userExists(username string) bool {
	data, err := os.ReadFile(UDB)
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

func getUserLine(username string) string {
	data, err := os.ReadFile(UDB)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, username+"|") {
			return line
		}
	}
	return ""
}

func uList() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📋  LIST SEMUA AKUN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	data, err := os.ReadFile(UDB)
	if err != nil || strings.TrimSpace(string(data)) == "" {
		warn("Belum ada akun terdaftar.")
		pause()
		return
	}

	today := time.Now().Format("2006-01-02")
	fmt.Printf("  %s┌────┬──────────────────┬────────────┬────────────┬──────────┬─────────┐%s\n", A1, NC)
	fmt.Printf("  %s│%s%s %-2s %s│%s%s %-16s %s│%s%s %-10s %s│%s%s %-10s %s│%s%s %-8s %s│%s%s %-7s %s│%s\n",
		A1, NC, BLD, "#", A1, NC, BLD, "Username", A1, NC, BLD, "Password", A1, NC, BLD, "Expired", A1, NC, BLD, "Kuota", A1, NC, BLD, "Status", A1, NC)
	fmt.Printf("  %s├────┼──────────────────┼────────────┼────────────┼──────────┼─────────┤%s\n", A1, NC)

	n := 1
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		u, p, e, q := parts[0], parts[1], parts[2], parts[3]
		sc, sl := LG, "AKTIF  "
		if e < today {
			sc = LR
			sl = "EXPIRED"
		}
		ql := "Unlim   "
		if q != "0" {
			ql = fmt.Sprintf("%-8s", q+"GB")
		}
		fmt.Printf("  %s│%s %s%-2d%s %s│%s %s%-16s%s %s│%s %s%-10s%s %s│%s %s%-10s%s %s│%s %-8s %s│%s %s%-7s%s %s│%s\n",
			A1, NC, DIM, n, NC, A1, NC, W, u, NC, A1, NC, A3, p, NC, A1, NC, Y, e, NC, A1, NC, ql, A1, NC, sc, sl, NC, A1, NC)
		n++
	}
	fmt.Printf("  %s└────┴──────────────────┴────────────┴────────────┴──────────┴─────────┘%s\n", A1, NC)
	fmt.Println()
	fmt.Printf("  %s  Total: %d akun  │  Expired: %d akun%s\n", DIM, n-1, expCount(), NC)
	pause()
}

func uInfo() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔍  INFO DETAIL AKUN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	un := readLine(fmt.Sprintf("  %sUsername%s: ", A3, NC))
	ln := getUserLine(un)
	if ln == "" {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}
	parts := strings.Split(ln, "|")
	u, p, e, q := parts[0], parts[1], parts[2], parts[3]
	note := "-"
	if len(parts) > 4 {
		note = parts[4]
	}

	domain := getDomain()
	port := getPort()
	ipPub := getIP()
	ql := "Unlimited"
	if q != "0" {
		ql = q + " GB"
	}
	maxl := getMaxlogin(u)
	if maxl == "" {
		maxl = "2"
	}
	showAkunBox(u, p, domain, port, ql, e, note, ipPub, maxl)
	pause()
}

func uDel() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🗑️   HAPUS AKUN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	data, err := os.ReadFile(UDB)
	if err != nil || strings.TrimSpace(string(data)) == "" {
		warn("Tidak ada akun.")
		pause()
		return
	}

	n := 1
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			fmt.Printf("  %s%3d.%s  %s%-22s%s  %sexp: %s%s\n", DIM, n, NC, W, parts[0], NC, DIM, parts[2], NC)
		}
		n++
	}
	fmt.Println()

	du := readLine(fmt.Sprintf("  %sUsername yang dihapus%s: ", A3, NC))
	if !userExists(du) {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}
	removeUserFromDB(du)
	delMaxlogin(du)
	reloadPW()
	tgSend(fmt.Sprintf("🗑 <b>Akun Dihapus</b> : <code>%s</code>", du))
	ok(fmt.Sprintf("Akun '%s%s%s' berhasil dihapus.", W, du, NC))
	pause()
}

func uRenew() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔁  PERPANJANG AKUN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	ru := readLine(fmt.Sprintf("  %sUsername%s    : ", A3, NC))
	if !userExists(ru) {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}

	rdStr := readLine(fmt.Sprintf("  %sTambah hari%s : ", A3, NC))
	rd := 30
	if rdStr != "" {
		rd, _ = strconv.Atoi(rdStr)
	}

	ln := getUserLine(ru)
	parts := strings.Split(ln, "|")
	ce := parts[2]
	today := time.Now().Format("2006-01-02")
	if ce < today {
		ce = today
	}
	ceTime, _ := time.Parse("2006-01-02", ce)
	ne := ceTime.AddDate(0, 0, rd).Format("2006-01-02")

	// Update database
	data, _ := os.ReadFile(UDB)
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		lparts := strings.Split(line, "|")
		if len(lparts) >= 3 && lparts[0] == ru {
			lparts[2] = ne
			line = strings.Join(lparts, "|")
		}
		lines = append(lines, line)
	}
	os.WriteFile(UDB, []byte(strings.Join(lines, "\n")+"\n"), 0644)

	tgSend(fmt.Sprintf("🔁 <b>Akun Diperpanjang</b>\n👤 User     : <code>%s</code>\n📅 Expired  : <b>%s</b>  (+%d hari)", ru, ne, rd))

	fmt.Println()
	fmt.Printf("  %s┌──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s✔  Akun berhasil diperpanjang!%s                        %s│%s\n", A1, NC, LG, NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Username  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, ru, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Expired   %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, ne, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Tambahan  %s %s│%s  %s+%-40d%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, LG, rd, NC, A1, NC)
	fmt.Printf("  %s└──────────────┴───────────────────────────────────────────┘%s\n", A1, NC)
	pause()
}

func uChpass() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔑  GANTI PASSWORD%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	pu := readLine(fmt.Sprintf("  %sUsername%s           : ", A3, NC))
	if !userExists(pu) {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}

	pp := readLine(fmt.Sprintf("  %sPassword baru%s [auto]: ", A3, NC))
	if pp == "" {
		pp = randPass()
	}

	data, _ := os.ReadFile(UDB)
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		lparts := strings.Split(line, "|")
		if len(lparts) >= 2 && lparts[0] == pu {
			lparts[1] = pp
			line = strings.Join(lparts, "|")
		}
		lines = append(lines, line)
	}
	os.WriteFile(UDB, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	reloadPW()

	fmt.Println()
	fmt.Printf("  %s┌──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s✔  Password berhasil diubah!%s                          %s│%s\n", A1, NC, LG, NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Username  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, pu, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Password  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, A3, pp, NC, A1, NC)
	fmt.Printf("  %s└──────────────┴───────────────────────────────────────────┘%s\n", A1, NC)
	pause()
}

func uTrial() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🎁  BUAT AKUN TRIAL%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	tu := "trial" + func() string {
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
		b := make([]byte, 6)
		for i := range b {
			b[i] = chars[rand.Intn(len(chars))]
		}
		return string(b)
	}()
	tp := randPass()
	te := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	ipPub := getIP()

	f, _ := os.OpenFile(UDB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	fmt.Fprintf(f, "%s|%s|%s|1|TRIAL\n", tu, tp, te)
	f.Close()
	reloadPW()

	domain := getDomain()
	port := getPort()
	tgSend(fmt.Sprintf("🎁 <b>Akun Trial Dibuat</b>\n👤 User  : <code>%s</code>\n🔑 Pass  : <code>%s</code>\n🖥 IP    : <code>%s</code>\n📅 Exp   : %s  (1 hari / 1 GB)", tu, tp, ipPub, te))
	showAkunBox(tu, tp, domain, port, "1 GB", te, "TRIAL", ipPub, "2")
	pause()
}

func uClean() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🧹  HAPUS AKUN EXPIRED%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	today := time.Now().Format("2006-01-02")
	data, err := os.ReadFile(UDB)
	if err != nil {
		inf("Tidak ada akun expired.")
		pause()
		return
	}

	cnt := 0
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 3 && parts[2] < today {
			ok(fmt.Sprintf("Dihapus: %s%s%s  %s(exp: %s)%s", W, parts[0], NC, DIM, parts[2], NC))
			delMaxlogin(parts[0])
			cnt++
		} else {
			kept = append(kept, line)
		}
	}
	os.WriteFile(UDB, []byte(strings.Join(kept, "\n")+"\n"), 0644)
	fmt.Println()
	if cnt > 0 {
		reloadPW()
		ok(fmt.Sprintf("Total %s%d%s akun expired dihapus.", W, cnt, NC))
	} else {
		inf("Tidak ada akun expired.")
	}
	pause()
}

func uMaxlogin() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔒  SET MAXLOGIN DEVICE%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	data, err := os.ReadFile(UDB)
	if err != nil || strings.TrimSpace(string(data)) == "" {
		warn("Belum ada akun terdaftar.")
		pause()
		return
	}

	fmt.Println()
	fmt.Printf("  %s┌──────────────────────────────────────────────────────┐%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s%-20s  %-8s  %-10s%s  %s│%s\n", A1, NC, DIM, "Username", "MaxDev", "Status", NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	today := time.Now().Format("2006-01-02")
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		u, e := parts[0], parts[2]
		ml := getMaxlogin(u)
		if ml == "" {
			ml = "-"
		}
		sc, sl := LG, "AKTIF  "
		if e < today {
			sc = LR
			sl = "EXPIRED"
		}
		fmt.Printf("  %s│%s  %s%-20s%s  %s%-8s%s  %s%-10s%s  %s│%s\n", A1, NC, W, u, NC, Y, ml, NC, sc, sl, NC, A1, NC)
	}
	fmt.Printf("  %s└──────────────────────────────────────────────────────┘%s\n", A1, NC)
	fmt.Println()

	mu := readLine(fmt.Sprintf("  %sUsername%s          : ", A3, NC))
	if !userExists(mu) {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}
	curML := getMaxlogin(mu)
	if curML == "" {
		curML = "2"
	}
	fmt.Printf("  %sMaxLogin saat ini : %s%s device%s\n", DIM, Y, curML, NC)
	nml := readLine(fmt.Sprintf("  %sMax Login Device%s [%s]: ", A3, NC, curML))
	if nml == "" {
		nml = curML
	}
	if _, err := strconv.Atoi(nml); err != nil {
		nml = curML
	}
	setMaxlogin(mu, nml)

	fmt.Println()
	fmt.Printf("  %s┌──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s✔  MaxLogin berhasil diatur!%s                          %s│%s\n", A1, NC, LG, NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Username  %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, mu, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Max Dev   %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, nml+" device", NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Info      %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, DIM, "Auto-delete jika melebihi limit", NC, A1, NC)
	fmt.Printf("  %s└──────────────┴───────────────────────────────────────────┘%s\n", A1, NC)

	// Setup cron
	cronLine := "*/5 * * * * /usr/local/bin/ogh-ziv --check-maxlogin >/dev/null 2>&1"
	crontabOut, _ := exec.Command("crontab", "-l").Output()
	if !strings.Contains(string(crontabOut), "check-maxlogin") {
		newCron := string(crontabOut) + cronLine + "\n"
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(newCron)
		cmd.Run()
	}
	pause()
}

// ════════════════════════════════════════════════════════════
//  JUALAN
// ════════════════════════════════════════════════════════════

func tAkun() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📨  TEMPLATE PESAN AKUN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	tu := readLine(fmt.Sprintf("  %sUsername%s: ", A3, NC))
	ln := getUserLine(tu)
	if ln == "" {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}
	parts := strings.Split(ln, "|")
	u, p, e, q := parts[0], parts[1], parts[2], parts[3]
	note := "-"
	if len(parts) > 4 {
		note = parts[4]
	}
	domain := getDomain()
	port := getPort()
	ipPub := getIP()
	ql := "Unlimited"
	if q != "0" {
		ql = q + " GB"
	}
	showAkunBox(u, p, domain, port, ql, e, note, ipPub, "2")
	pause()
}

func setStore() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s⚙️   PENGATURAN TOKO%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	curBrand, curTG := "OGH-ZIV", "-"
	if data, err := os.ReadFile(STRF); err == nil {
		conf := parseConf(string(data))
		if b := conf["BRAND"]; b != "" {
			curBrand = b
		}
		if t := conf["ADMIN_TG"]; t != "" {
			curTG = t
		}
	}

	ib := readLine(fmt.Sprintf("  %sNama Brand%s [%s]   : ", A3, NC, curBrand))
	if ib == "" {
		ib = curBrand
	}
	it := readLine(fmt.Sprintf("  %sUsername TG Admin%s [%s]: ", A3, NC, curTG))
	if it == "" {
		it = curTG
	}
	os.WriteFile(STRF, []byte(fmt.Sprintf("BRAND=%s\nADMIN_TG=%s\n", ib, it)), 0644)
	ok("Pengaturan toko disimpan!")
	pause()
}

// ════════════════════════════════════════════════════════════
//  TELEGRAM BOT
// ════════════════════════════════════════════════════════════

func tgSetup() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🤖  SETUP BOT TELEGRAM%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	inf(fmt.Sprintf("Buka %s@BotFather%s di Telegram → ketik /newbot → salin TOKEN", A3, NC))
	inf("Kirim /start ke bot → buka URL:")
	fmt.Printf("  %s     api.telegram.org/bot<TOKEN>/getUpdates%s\n\n", DIM, NC)

	tok := readLine(fmt.Sprintf("  %sBot Token%s     : ", A3, NC))
	if tok == "" {
		errMsg("Token kosong!")
		pause()
		return
	}
	cid := readLine(fmt.Sprintf("  %sChat ID Admin%s : ", A3, NC))
	if cid == "" {
		errMsg("Chat ID kosong!")
		pause()
		return
	}

	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", tok))
	if err != nil {
		errMsg("Gagal terhubung ke Telegram!")
		pause()
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, `"ok":true`) {
		errMsg("Token tidak valid atau tidak bisa terhubung!")
		pause()
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	bname := ""
	if r, ok := result["result"].(map[string]interface{}); ok {
		if u, ok := r["username"].(string); ok {
			bname = u
		}
	}

	os.WriteFile(BOTF, []byte(fmt.Sprintf("BOT_TOKEN=%s\nCHAT_ID=%s\nBOT_NAME=%s\n", tok, cid, bname)), 0644)
	tgRaw(tok, cid, "✅ <b>OGH-ZIV Premium</b> bot terhubung ke server VPS!")

	fmt.Println()
	fmt.Printf("  %s┌──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s✔  Bot Telegram berhasil terhubung!%s                   %s│%s\n", A1, NC, LG, NC, A1, NC)
	fmt.Printf("  %s├──────────────┼───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Bot Name  %s %s│%s  %s@%-40s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, bname, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Chat ID   %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, cid, NC, A1, NC)
	fmt.Printf("  %s└──────────────┴───────────────────────────────────────────┘%s\n", A1, NC)
	pause()
}

func tgStatus() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📡  STATUS BOT TELEGRAM%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	if _, err := os.Stat(BOTF); os.IsNotExist(err) {
		warn("Bot belum dikonfigurasi.")
		a := readLine("  Setup sekarang? [y/N]: ")
		if strings.ToLower(a) == "y" {
			tgSetup()
		}
		return
	}

	data, _ := os.ReadFile(BOTF)
	conf := parseConf(string(data))
	token := conf["BOT_TOKEN"]
	botName := conf["BOT_NAME"]
	chatID := conf["CHAT_ID"]

	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token))
	if err != nil || resp == nil {
		errMsg("Bot tidak dapat terhubung. Cek token!")
		pause()
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), `"ok":true`) {
		errMsg("Bot tidak dapat terhubung. Cek token!")
		pause()
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	fn := ""
	if r, ok := result["result"].(map[string]interface{}); ok {
		if n, ok := r["first_name"].(string); ok {
			fn = n
		}
	}

	fmt.Println()
	fmt.Printf("  %s╔══════════════════════════════════════════════════════════╗%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s🟢  Bot Aktif & Terhubung%s                              %s║%s\n", A1, NC, LG, NC, A1, NC)
	fmt.Printf("  %s╠══════════════╦═══════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s %s Nama      %s %s║%s  %s%-41s%s  %s║%s\n", A1, NC, DIM, NC, A1, NC, W, fn, NC, A1, NC)
	fmt.Printf("  %s║%s %s Username  %s %s║%s  %s@%-40s%s  %s║%s\n", A1, NC, DIM, NC, A1, NC, W, botName, NC, A1, NC)
	fmt.Printf("  %s║%s %s Chat ID   %s %s║%s  %s%-41s%s  %s║%s\n", A1, NC, DIM, NC, A1, NC, Y, chatID, NC, A1, NC)
	fmt.Printf("  %s╚══════════════╩═══════════════════════════════════════════╝%s\n", A1, NC)
	fmt.Println()
	ts := readLine(fmt.Sprintf("  %sKirim pesan test?%s [y/N]: ", A3, NC))
	if strings.ToLower(ts) == "y" {
		tgSend("🟢 <b>Test OGH-ZIV Premium</b> — Bot berjalan normal! ✅")
		ok("Pesan test dikirim ke Telegram!")
	}
	pause()
}

func tgKirimAkun() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📤  KIRIM AKUN KE TELEGRAM%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	if _, err := os.Stat(BOTF); os.IsNotExist(err) {
		errMsg("Bot belum dikonfigurasi!")
		pause()
		return
	}

	botData, _ := os.ReadFile(BOTF)
	botConf := parseConf(string(botData))
	token := botConf["BOT_TOKEN"]
	defaultChatID := botConf["CHAT_ID"]

	brand := "OGH-ZIV"
	if storeData, err := os.ReadFile(STRF); err == nil {
		if b := parseConf(string(storeData))["BRAND"]; b != "" {
			brand = b
		}
	}

	su := readLine(fmt.Sprintf("  %sUsername akun%s    : ", A3, NC))
	ln := getUserLine(su)
	if ln == "" {
		errMsg("User tidak ditemukan!")
		pause()
		return
	}
	parts := strings.Split(ln, "|")
	u, p, e, q := parts[0], parts[1], parts[2], parts[3]

	did := readLine(fmt.Sprintf("  %sChat ID tujuan%s [%s]: ", A3, NC, defaultChatID))
	if did == "" {
		did = defaultChatID
	}

	domain := getDomain()
	port := getPort()
	ipPub := getIP()
	ql := "Unlimited"
	if q != "0" {
		ql = q + " GB"
	}

	expTime, _ := time.Parse("2006-01-02", e)
	sisa := int(time.Until(expTime).Hours() / 24)
	sisaStr := "Expired"
	if sisa >= 0 {
		sisaStr = fmt.Sprintf("%d hari lagi", sisa)
	}

	msg := fmt.Sprintf("🔒 <b>%s — Akun VPN UDP Premium</b>\n\n┌────────────────────────────\n│ 👤 <b>Username</b>  : <code>%s</code>\n│ 🔑 <b>Password</b>  : <code>%s</code>\n├────────────────────────────\n│ 🖥 <b>IP Publik</b>  : <code>%s</code>\n│ 🌐 <b>Host</b>      : <code>%s</code>\n│ 🔌 <b>Port</b>      : <code>%s</code>\n│ 📡 <b>Obfs</b>      : <code>zivpn</code>\n├────────────────────────────\n│ 📦 <b>Kuota</b>     : %s\n│ 📅 <b>Expired</b>   : %s\n│ ⏳ <b>Sisa</b>      : %s\n└────────────────────────────\n\n📱 Download <b>ZiVPN</b> di Play Store / App Store\n⚠️ Jangan share akun ini ke orang lain!",
		brand, u, p, ipPub, domain, port, ql, e, sisaStr)

	resp, err := http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
		url.Values{"chat_id": {did}, "text": {msg}, "parse_mode": {"HTML"}},
	)
	fmt.Println()
	if err == nil && resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), `"ok":true`) {
			ok(fmt.Sprintf("Akun '%s%s%s' berhasil dikirim ke Telegram!", W, u, NC))
		} else {
			errMsg("Gagal kirim! Periksa Chat ID atau token.")
		}
	} else {
		errMsg("Gagal kirim! Periksa Chat ID atau token.")
	}
	pause()
}

func tgBroadcast() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📢  BROADCAST PESAN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	if _, err := os.Stat(BOTF); os.IsNotExist(err) {
		errMsg("Bot belum dikonfigurasi!")
		pause()
		return
	}
	data, _ := os.ReadFile(BOTF)
	conf := parseConf(string(data))

	fmt.Printf("  %sKetik pesan. Ketik %sSELESAI%s%s di baris baru untuk kirim.%s\n\n", DIM, W, DIM, DIM, NC)

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "SELESAI" {
			break
		}
		lines = append(lines, line)
	}
	msg := strings.Join(lines, "\n")
	if msg == "" {
		errMsg("Pesan kosong!")
		pause()
		return
	}
	http.PostForm(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", conf["BOT_TOKEN"]),
		url.Values{"chat_id": {conf["CHAT_ID"]}, "text": {msg}},
	)
	ok("Broadcast berhasil dikirim!")
	pause()
}

func tgGuide() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📖  PANDUAN BUAT BOT TELEGRAM%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	fmt.Printf("  %s╔══════════════════════════════════════════════════════════╗%s\n", A1, NC)
	fmt.Printf("  %s║%s  %sLANGKAH 1 — Buat Bot di BotFather%s                      %s║%s\n", A1, NC, Y, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s1.%s Buka Telegram → cari %s@BotFather%s                       %s║%s\n", A1, NC, W, NC, A3, NC, A1, NC)
	fmt.Printf("  %s║%s  %s2.%s Kirim perintah %s/newbot%s                               %s║%s\n", A1, NC, W, NC, Y, NC, A1, NC)
	fmt.Printf("  %s║%s  %s3.%s Masukkan nama bot → contoh: %sOGH ZIV VPN%s              %s║%s\n", A1, NC, W, NC, W, NC, A1, NC)
	fmt.Printf("  %s║%s  %s4.%s Masukkan username (akhiran %sbot%s)                       %s║%s\n", A1, NC, W, NC, Y, NC, A1, NC)
	fmt.Printf("  %s║%s  %s5.%s Salin %sTOKEN%s yang diberikan BotFather               %s║%s\n", A1, NC, W, NC, Y, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %sLANGKAH 2 — Ambil Chat ID%s                               %s║%s\n", A1, NC, Y, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s1.%s Kirim %s/start%s ke bot kamu di Telegram               %s║%s\n", A1, NC, W, NC, Y, NC, A1, NC)
	fmt.Printf("  %s║%s  %s2.%s Buka: %sapi.telegram.org/bot<TOKEN>/getUpdates%s       %s║%s\n", A1, NC, W, NC, DIM, NC, A1, NC)
	fmt.Printf("  %s║%s  %s3.%s Cari nilai %s\"id\"%s di bagian %s\"from\"%s                  %s║%s\n", A1, NC, W, NC, Y, NC, Y, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %sLANGKAH 3 — Hubungkan ke OGH-ZIV%s                        %s║%s\n", A1, NC, Y, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %s1.%s Menu Telegram → %s[1] Setup / Konfigurasi Bot%s         %s║%s\n", A1, NC, W, NC, A3, NC, A1, NC)
	fmt.Printf("  %s║%s  %s2.%s Masukkan Token dan Chat ID                           %s║%s\n", A1, NC, W, NC, A1, NC)
	fmt.Printf("  %s║%s  %s3.%s %s✅ Selesai! Notifikasi otomatis aktif%s              %s║%s\n", A1, NC, W, NC, LG, NC, A1, NC)
	fmt.Printf("  %s╠══════════════════════════════════════════════════════════╣%s\n", A1, NC)
	fmt.Printf("  %s║%s  %shttps://t.me/BotFather%s                                   %s║%s\n", A1, NC, A3, NC, A1, NC)
	fmt.Printf("  %s╚══════════════════════════════════════════════════════════╝%s\n", A1, NC)
	pause()
}

// ════════════════════════════════════════════════════════════
//  SERVICE FUNCTIONS
// ════════════════════════════════════════════════════════════

func svcStatus() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🖥️   STATUS SERVICE%s", IT, AL, NC))
	boxBot()
	fmt.Println()
	cmd := exec.Command("systemctl", "status", "zivpn", "--no-pager", "-l")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	pause()
}

func svcBandwidth() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📊  BANDWIDTH / KONEKSI AKTIF%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	port := getPort()
	inf(fmt.Sprintf("Koneksi aktif ke port %s%s%s:", Y, port, NC))
	fmt.Println()

	cmd := exec.Command("bash", "-c", fmt.Sprintf("ss -u -n -p 2>/dev/null | grep ':%s'", port))
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) == "" {
		inf("Tidak ada koneksi UDP aktif saat ini.")
	} else {
		fmt.Print(string(out))
	}

	fmt.Println()
	inf("Statistik network interface:")
	devData, _ := os.ReadFile("/proc/net/dev")
	for i, line := range strings.Split(string(devData), "\n") {
		if i < 2 {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 10 {
			continue
		}
		iface := strings.TrimSuffix(parts[0], ":")
		if iface == "lo" || iface == "" {
			continue
		}
		fmt.Printf("  %-12s RX: %-12s TX: %s\n", iface, parts[1], parts[9])
	}
	pause()
}

func svcLog() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s📄  LOG ZIVPN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	if _, err := os.Stat(LOG); err == nil {
		cmd := exec.Command("tail", "-60", LOG)
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("journalctl", "-u", "zivpn", "-n", "60", "--no-pager")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	pause()
}

func svcPort() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔧  GANTI PORT%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	cp := getPort()
	fmt.Printf("  Port saat ini : %s%s%s\n", Y, cp, NC)
	np := readLine(fmt.Sprintf("  %sPort baru%s     : ", A3, NC))

	nPortInt, err := strconv.Atoi(np)
	if err != nil || nPortInt < 1 || nPortInt > 65535 {
		errMsg("Port tidak valid!")
		pause()
		return
	}

	// Update config
	cfgData, _ := os.ReadFile(CFG)
	var cfgMap map[string]interface{}
	json.Unmarshal(cfgData, &cfgMap)
	cfgMap["listen"] = ":" + np
	out, _ := json.MarshalIndent(cfgMap, "", "  ")
	os.WriteFile(CFG, out, 0644)

	// UFW
	exec.Command("bash", "-c", fmt.Sprintf("command -v ufw &>/dev/null && { ufw delete allow %s/udp &>/dev/null; ufw allow %s/udp &>/dev/null; }", cp, np)).Run()
	// iptables
	exec.Command("iptables", "-D", "INPUT", "-p", "udp", "--dport", cp, "-j", "ACCEPT").Run()
	exec.Command("iptables", "-I", "INPUT", "-p", "udp", "--dport", np, "-j", "ACCEPT").Run()
	exec.Command("systemctl", "restart", "zivpn").Run()

	ok(fmt.Sprintf("Port diubah: %s%s%s → %s%s%s", Y, cp, NC, LG, np, NC))
	pause()
}

func svcBackup() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s💾  BACKUP DATA%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	bfile := fmt.Sprintf("/root/oghziv-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
	inf(fmt.Sprintf("Membuat backup → %s%s%s", W, bfile, NC))
	cmd := exec.Command("tar", "-czf", bfile, DIR)
	if cmd.Run() == nil {
		ok(fmt.Sprintf("Backup berhasil: %s%s%s", W, bfile, NC))
	} else {
		errMsg("Backup gagal!")
	}
	pause()
}

func svcRestore() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s♻️   RESTORE DATA%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	bpath := readLine(fmt.Sprintf("  %sPath file backup (.tar.gz)%s: ", A3, NC))
	if _, err := os.Stat(bpath); os.IsNotExist(err) {
		errMsg("File tidak ditemukan!")
		pause()
		return
	}
	warn("Restore akan menimpa semua data saat ini!")
	cf := readLine("  Lanjutkan? [y/N]: ")
	if strings.ToLower(cf) != "y" {
		inf("Dibatalkan.")
		pause()
		return
	}
	exec.Command("tar", "-xzf", bpath, "-C", "/").Run()
	reloadPW()
	ok("Restore berhasil!")
	pause()
}

// ════════════════════════════════════════════════════════════
//  DOMAIN MANAGEMENT
// ════════════════════════════════════════════════════════════

func domainSet() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s✏️   SET / GANTI DOMAIN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	cur := getDomain()
	ip := getIP()
	fmt.Printf("  %sDomain saat ini : %s%s%s\n", DIM, W, cur, NC)
	fmt.Printf("  %sIP Publik VPS   : %s%s%s\n\n", DIM, A3, ip, NC)
	inf(fmt.Sprintf("Pastikan DNS domain sudah diarahkan ke IP: %s%s%s\n", Y, ip, NC))

	nd := readLine(fmt.Sprintf("  %sDomain baru%s (kosongkan = pakai IP): ", A3, NC))
	if nd == "" {
		os.WriteFile(DOMF, []byte(ip), 0644)
		ok(fmt.Sprintf("Domain diatur ke IP publik: %s%s%s", A3, ip, NC))
	} else {
		os.WriteFile(DOMF, []byte(nd), 0644)
		ok(fmt.Sprintf("Domain disimpan: %s%s%s", W, nd, NC))
		rs := readLine(fmt.Sprintf("  %sRegenerasi SSL sekarang?%s [y/N]: ", A3, NC))
		if strings.ToLower(rs) == "y" {
			domainSSL()
		}
	}
	pause()
}

func domainUseIP() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔄  GUNAKAN IP PUBLIK%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	ip := getIP()
	os.WriteFile(DOMF, []byte(ip), 0644)
	ok(fmt.Sprintf("Domain direset ke IP publik: %s%s%s", A3, ip, NC))
	pause()
}

func domainCheck() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔍  CEK DNS DOMAIN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	dom := getDomain()
	ip := getIP()
	fmt.Printf("  %sDomain  : %s%s%s\n", DIM, W, dom, NC)
	fmt.Printf("  %sIP VPS  : %s%s%s\n\n", DIM, A3, ip, NC)
	inf("Resolving DNS...")

	cmd := exec.Command("bash", "-c", fmt.Sprintf(`host %s 2>/dev/null | grep "has address" | awk '{print $NF}' | head -1`, dom))
	out, _ := cmd.Output()
	resolved := strings.TrimSpace(string(out))

	if resolved == "" {
		cmd2 := exec.Command("bash", "-c", fmt.Sprintf(`nslookup %s 2>/dev/null | awk '/^Address:/{print $2}' | grep -v '#' | head -1`, dom))
		out2, _ := cmd2.Output()
		resolved = strings.TrimSpace(string(out2))
	}

	if resolved == "" {
		errMsg(fmt.Sprintf("Tidak dapat meresolve domain: %s%s%s", W, dom, NC))
	} else if resolved == ip {
		ok(fmt.Sprintf("DNS OK — %s%s%s → %s%s%s %s(cocok dengan IP VPS)%s", W, dom, NC, A3, resolved, NC, LG, NC))
	} else {
		warn("DNS mismatch!")
		fmt.Printf("  %sDomain resolve ke : %s%s%s\n", DIM, LR, resolved, NC)
		fmt.Printf("  %sIP VPS            : %s%s%s\n", DIM, A3, ip, NC)
		inf(fmt.Sprintf("Arahkan DNS domain ke IP: %s%s%s", Y, ip, NC))
	}
	pause()
}

func domainSSL() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🔄  REGENERASI SSL CERTIFICATE%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	dom := getDomain()
	inf(fmt.Sprintf("Membuat SSL baru untuk: %s%s%s", W, dom, NC))

	cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-newkey", "ec",
		"-pkeyopt", "ec_paramgen_curve:P-256",
		"-keyout", DIR+"/zivpn.key",
		"-out", DIR+"/zivpn.crt",
		"-subj", "/CN="+dom,
		"-days", "3650")
	if cmd.Run() == nil {
		ok(fmt.Sprintf("SSL Certificate (10 tahun) berhasil dibuat untuk %s%s%s", W, dom, NC))
	} else {
		errMsg("Gagal generate SSL!")
		pause()
		return
	}
	exec.Command("systemctl", "restart", "zivpn").Run()
	ok("Service direstart dengan SSL baru.")
	pause()
}

// ════════════════════════════════════════════════════════════
//  INSTALL & UNINSTALL
// ════════════════════════════════════════════════════════════

func doInstall() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s🚀  INSTALL ZIVPN%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	inf("Membersihkan file lama (jika ada)...")
	exec.Command("systemctl", "stop", "zivpn.service").Run()
	exec.Command("systemctl", "disable", "zivpn.service").Run()
	os.Remove(BIN)
	os.Remove(SVC)
	os.Remove(DIR + "/zivpn.key")
	os.Remove(DIR + "/zivpn.crt")
	os.Remove(CFG)
	os.Remove(LOG)
	exec.Command("systemctl", "daemon-reload").Run()
	ok("File lama dibersihkan — data akun & konfigurasi dipertahankan")

	sip := getIP()
	inpDomain := readLine(fmt.Sprintf("  %sDomain / IP%s            : ", A3, NC))
	if inpDomain == "" {
		inpDomain = sip
	}
	inpPort := readLine(fmt.Sprintf("  %sPort%s [5667]             : ", A3, NC))
	if inpPort == "" {
		inpPort = "5667"
	}
	inpBrand := readLine(fmt.Sprintf("  %sNama Brand / Toko%s       : ", A3, NC))
	if inpBrand == "" {
		inpBrand = "OGH-ZIV"
	}
	inpTG := readLine(fmt.Sprintf("  %sUsername Telegram Admin%s  : ", A3, NC))
	if inpTG == "" {
		inpTG = "-"
	}

	fmt.Println()
	fmt.Printf("  %s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", A1, NC)
	inf(fmt.Sprintf("Memulai instalasi %s%sOGH-ZIV Premium%s...", AL, "", NC))
	fmt.Printf("  %s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n\n", A1, NC)

	// Dependensi
	inf("Menginstall dependensi...")
	exec.Command("apt-get", "update", "-qq").Run()
	exec.Command("apt-get", "install", "-y", "-qq", "curl", "wget", "openssl",
		"python3", "iptables", "iptables-persistent", "netfilter-persistent").Run()
	ok("Dependensi terpasang")

	// Direktori
	os.MkdirAll(DIR, 0755)
	os.Create(UDB)
	os.Create(LOG)
	os.WriteFile(DOMF, []byte(inpDomain), 0644)
	os.WriteFile(THEMEF, []byte("7"), 0644)
	os.WriteFile(STRF, []byte(fmt.Sprintf("BRAND=%s\nADMIN_TG=%s\n", inpBrand, inpTG)), 0644)
	ok("Direktori & konfigurasi dibuat")

	// Download binary
	fmt.Printf("\n  %s──────────────────────────────────────────────%s\n", A1, NC)
	inf("Downloading UDP Service...")
	cmd := exec.Command("wget", BINARY_URL, "-O", BIN)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	info, err := os.Stat(BIN)
	if err != nil || info.Size() == 0 {
		errMsg("Gagal download binary ZiVPN!")
		fmt.Printf("  %sCoba jalankan manual:%s\n", Y, NC)
		fmt.Printf("  %swget %s -O %s%s\n", W, BINARY_URL, BIN, NC)
		os.Remove(BIN)
		pause()
		return
	}
	os.Chmod(BIN, 0755)
	ok(fmt.Sprintf("Binary ZiVPN siap (%d KB)", info.Size()/1024))

	// Download config
	inf("Mengunduh config.json...")
	cmd2 := exec.Command("wget", CONFIG_URL, "-O", CFG)
	cmd2.Run()

	cfgInfo, err := os.Stat(CFG)
	if err != nil || cfgInfo.Size() == 0 {
		warn("config.json tidak bisa diunduh, membuat manual...")
		cfgContent := fmt.Sprintf(`{
  "listen": ":%s",
  "cert": "/etc/zivpn/zivpn.crt",
  "key": "/etc/zivpn/zivpn.key",
  "obfs": "zivpn",
  "auth": {
    "mode": "passwords",
    "config": []
  }
}`, inpPort)
		os.WriteFile(CFG, []byte(cfgContent), 0644)
	} else {
		// Update port
		cfgData, _ := os.ReadFile(CFG)
		var cfgMap map[string]interface{}
		if json.Unmarshal(cfgData, &cfgMap) == nil {
			cfgMap["listen"] = ":" + inpPort
			out, _ := json.MarshalIndent(cfgMap, "", "  ")
			os.WriteFile(CFG, out, 0644)
		}
	}
	ok("config.json siap")

	// SSL
	fmt.Printf("\n  %s──────────────────────────────────────────────%s\n", A1, NC)
	inf("Generating cert files...")
	sslCmd := exec.Command("openssl", "req", "-new", "-newkey", "rsa:4096",
		"-days", "365", "-nodes", "-x509",
		"-subj", "/C=US/ST=California/L=Los Angeles/O=Example Corp/OU=IT Department/CN=zivpn",
		"-keyout", DIR+"/zivpn.key",
		"-out", DIR+"/zivpn.crt")
	sslCmd.Run()
	ok("SSL Certificate RSA-4096 (1 tahun) dibuat")

	// Kernel buffer
	exec.Command("sysctl", "-w", "net.core.rmem_max=16777216").Run()
	exec.Command("sysctl", "-w", "net.core.wmem_max=16777216").Run()
	ok("Buffer UDP dioptimasi (rmem/wmem 16MB)")

	// Systemd service
	fmt.Printf("\n  %s──────────────────────────────────────────────%s\n", A1, NC)
	inf("Membuat systemd service...")
	svcContent := fmt.Sprintf(`[Unit]
Description=zivpn VPN Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s server -c %s
Restart=always
RestartSec=3
Environment=ZIVPN_LOG_LEVEL=info
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
NoNewPrivileges=true
LimitNOFILE=1048576
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=multi-user.target
`, DIR, BIN, CFG, LOG, LOG)
	os.WriteFile(SVC, []byte(svcContent), 0644)
	ok("Systemd service dibuat")

	// iptables
	fmt.Printf("\n  %s──────────────────────────────────────────────%s\n", A1, NC)
	inf("Mengatur iptables & UDP port forwarding...")
	ifaceCmd := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1")
	ifaceOut, _ := ifaceCmd.Output()
	iface := strings.TrimSpace(string(ifaceOut))

	// Clean old rules
	for {
		cmd := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
			"-i", iface, "-p", "udp", "--dport", "6000:19999",
			"-j", "DNAT", "--to-destination", ":"+inpPort)
		if cmd.Run() != nil {
			break
		}
	}
	exec.Command("iptables", "-t", "nat", "-A", "PREROUTING",
		"-i", iface, "-p", "udp", "--dport", "6000:19999",
		"-j", "DNAT", "--to-destination", ":"+inpPort).Run()
	exec.Command("iptables", "-A", "FORWARD", "-p", "udp", "-d", "127.0.0.1", "--dport", inpPort, "-j", "ACCEPT").Run()
	exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "127.0.0.1/32", "-o", iface, "-j", "MASQUERADE").Run()
	exec.Command("netfilter-persistent", "save").Run()
	ok(fmt.Sprintf("IPTables: UDP 6000-19999 → %s via %s", inpPort, iface))

	// UFW
	exec.Command("bash", "-c", fmt.Sprintf("command -v ufw &>/dev/null && { ufw allow 6000:19999/udp; ufw allow %s/udp; }", inpPort)).Run()
	exec.Command("iptables", "-I", "INPUT", "-p", "udp", "--dport", inpPort, "-j", "ACCEPT").Run()

	// Start service
	fmt.Printf("\n  %s──────────────────────────────────────────────%s\n", A1, NC)
	inf("Mengaktifkan service ZiVPN...")
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "zivpn.service").Run()
	exec.Command("systemctl", "start", "zivpn.service").Run()
	time.Sleep(1 * time.Second)
	if isUp() {
		ok("Service ZiVPN aktif & berjalan")
	} else {
		warn("Service gagal start — cek: journalctl -u zivpn -n 20")
	}

	setupMenuCmd()

	// Summary
	fmt.Println()
	fmt.Printf("  %s┌──────────────────────────────────────────────────────────┐%s\n", A1, NC)
	fmt.Printf("  %s│%s  %s%s  ✦ OGH-ZIV PREMIUM BERHASIL DIINSTALL!%s                %s│%s\n", A1, NC, LG, BLD, NC, A1, NC)
	fmt.Printf("  %s├──────────────┬───────────────────────────────────────────┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Domain    %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, inpDomain, NC, A1, NC)
	fmt.Printf("  %s│%s %s Port      %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, Y, inpPort, NC, A1, NC)
	fmt.Printf("  %s│%s %s Brand     %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, AL, inpBrand, NC, A1, NC)
	fmt.Printf("  %s├╌╌╌╌╌╌╌╌╌╌╌╌╌╌┼╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┤%s\n", A1, NC)
	fmt.Printf("  %s│%s %s Forwarding%s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, "UDP 6000-19999 → "+inpPort, NC, A1, NC)
	fmt.Printf("  %s│%s %s Interface %s %s│%s  %s%-41s%s  %s│%s\n", A1, NC, DIM, NC, A1, NC, W, iface, NC, A1, NC)
	fmt.Printf("  %s└──────────────┴───────────────────────────────────────────┘%s\n", A1, NC)
	fmt.Println()
	fmt.Printf("  %sKetik %smenu%s%s untuk membuka panel kapan saja.%s\n", DIM, A1, NC, DIM, NC)
	fmt.Println()
	pause()
}

func doUninstall() {
	showHeader()
	boxTop()
	boxBtn(fmt.Sprintf("  %s%s⚠️   UNINSTALL OGH-ZIV%s", IT, AL, NC))
	boxBot()
	fmt.Println()

	warn("Semua data user & konfigurasi akan DIHAPUS PERMANEN!")
	cf := readLine(fmt.Sprintf("  %sKetik 'HAPUS' untuk konfirmasi%s: ", LR, NC))
	if cf != "HAPUS" {
		inf("Dibatalkan.")
		pause()
		return
	}

	exec.Command("systemctl", "stop", "zivpn.service").Run()
	exec.Command("systemctl", "disable", "zivpn.service").Run()
	os.Remove(SVC)
	os.Remove(BIN)
	os.RemoveAll(DIR)
	exec.Command("systemctl", "daemon-reload").Run()

	ifaceCmd := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1")
	ifaceOut, _ := ifaceCmd.Output()
	iface := strings.TrimSpace(string(ifaceOut))

	for {
		cmd := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
			"-i", iface, "-p", "udp", "--dport", "6000:19999",
			"-j", "DNAT", "--to-destination", ":5667")
		if cmd.Run() != nil {
			break
		}
	}
	exec.Command("iptables", "-D", "FORWARD", "-p", "udp", "-d", "127.0.0.1", "--dport", "5667", "-j", "ACCEPT").Run()
	exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-s", "127.0.0.1/32", "-o", iface, "-j", "MASQUERADE").Run()
	exec.Command("netfilter-persistent", "save").Run()

	os.Remove("/usr/local/bin/menu")
	os.Remove("/usr/local/bin/ogh-ziv")
	os.Remove("/etc/profile.d/ogh-ziv.sh")

	exec.Command("bash", "-c", "sed -i '/alias menu=/d' ~/.bashrc 2>/dev/null").Run()
	exec.Command("bash", "-c", "sed -i '/alias zivpn=/d' ~/.bashrc 2>/dev/null").Run()
	exec.Command("bash", "-c", "sed -i '/alias menu=/d' /root/.profile 2>/dev/null").Run()

	ok("OGH-ZIV Premium berhasil diuninstall sepenuhnya.")
	fmt.Printf("  %sSemua binary, service, data, iptables & menu telah dihapus.%s\n", DIM, NC)
	pause()
	os.Exit(0)
}

// ── SETUP MENU COMMAND ─────────────────────────────────────
func setupMenuCmd() {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "/usr/local/bin/ogh-ziv"
	}
	absPath, _ := filepath.Abs(execPath)

	if absPath != "/usr/local/bin/ogh-ziv" {
		exec.Command("cp", absPath, "/usr/local/bin/ogh-ziv").Run()
		os.Chmod("/usr/local/bin/ogh-ziv", 0755)
	}

	os.Remove("/usr/local/bin/menu")
	os.Symlink("/usr/local/bin/ogh-ziv", "/usr/local/bin/menu")
	os.Chmod("/usr/local/bin/menu", 0755)

	profileContent := "#!/bin/bash\nalias menu='bash /usr/local/bin/ogh-ziv'\nalias zivpn='bash /usr/local/bin/ogh-ziv'\n"
	os.WriteFile("/etc/profile.d/ogh-ziv.sh", []byte(profileContent), 0755)
}

// ════════════════════════════════════════════════════════════
//  MENUS
// ════════════════════════════════════════════════════════════

func menuTema() {
	for {
		exec.Command("clear").Run()
		loadTheme()
		curTheme := "1"
		if b, err := os.ReadFile(THEMEF); err == nil {
			curTheme = strings.TrimSpace(string(b))
		}
		fmt.Println()
		fmt.Printf("  %s╔══════════════════════════════════════════════════════╗%s\n", A1, NC)
		fmt.Printf("  %s║%s  %s%s  🎨  PILIH TEMA WARNA%s                           %s║%s\n", A1, NC, IT, AL, NC, A1, NC)
		fmt.Printf("  %s╠══════════════════════════════════════════════════════╣%s\n", A1, NC)
		fmt.Printf("  %s║%s                                                      %s║%s\n", A1, NC, A1, NC)

		themes := []struct{ icon, label string }{
			{"💜", "[1]  VIOLET  — Ungu Premium"},
			{"🩵", "[2]  CYAN    — Neon Biru"},
			{"💚", "[3]  GREEN   — Matrix Hijau"},
			{"💛", "[4]  GOLD    — Emas Mewah"},
			{"❤️", "[5]  RED     — Merah Elegan"},
			{"🩷", "[6]  PINK    — Pink Pastel"},
			{"🌈", "[7]  RAINBOW — Pelangi Cantik"},
		}

		for i, t := range themes {
			n := strconv.Itoa(i + 1)
			mark := "   "
			if curTheme == n {
				mark = A2 + "▶" + NC + " "
			}
			fmt.Printf("  %s║%s    %s%s  %s%-30s        %s║%s\n", A1, NC, mark, t.icon, A1+n+NC+"  ", t.label, A1, NC)
		}

		fmt.Printf("  %s║%s                                                      %s║%s\n", A1, NC, A1, NC)
		fmt.Printf("  %s╠══════════════════════════════════════════════════════╣%s\n", A1, NC)
		fmt.Printf("  %s║%s  %sTema aktif sekarang : %s%s%s                        %s║%s\n", A1, NC, DIM, AT, THEME_NAME, NC, A1, NC)
		fmt.Printf("  %s╠══════════════════════════════════════════════════════╣%s\n", A1, NC)
		fmt.Printf("  %s║%s  %s[0]%s  ◀  Kembali ke menu utama                      %s║%s\n", A1, NC, LR, NC, A1, NC)
		fmt.Printf("  %s╚══════════════════════════════════════════════════════╝%s\n", A1, NC)
		fmt.Println()

		ch := readLine(fmt.Sprintf("  %s›%s Pilih tema [0-7]: ", A1, NC))
		if n, err := strconv.Atoi(ch); err == nil && n >= 1 && n <= 7 {
			os.WriteFile(THEMEF, []byte(ch), 0644)
			loadTheme()
			ok(fmt.Sprintf("Tema %s%s%s aktif!", AT, THEME_NAME, NC))
			time.Sleep(800 * time.Millisecond)
		} else if ch == "0" {
			break
		} else {
			warn("Pilihan tidak valid!")
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func menuAkun() {
	for {
		showHeader()
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  👤  KELOLA AKUN USER%s", IT, AL, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  ➕  Tambah Akun Baru", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[2]%s  📋  List Semua Akun", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[3]%s  🔍  Detail Akun", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[4]%s  🗑️   Hapus Akun", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[5]%s  🔁  Perpanjang Akun", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[6]%s  🔑  Ganti Password", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[7]%s  🎁  Buat Akun Trial", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[8]%s  🧹  Hapus Akun Expired", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[9]%s  🔒  Set MaxLogin Device", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[0]%s  ◀   Kembali", LR, NC))
		boxBot()
		fmt.Println()

		ch := readLine(fmt.Sprintf("  %s›%s ", A1, NC))
		switch ch {
		case "1":
			uAdd()
		case "2":
			uList()
		case "3":
			uInfo()
		case "4":
			uDel()
		case "5":
			uRenew()
		case "6":
			uChpass()
		case "7":
			uTrial()
		case "8":
			uClean()
		case "9":
			uMaxlogin()
		case "0":
			return
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

func menuJualan() {
	for {
		showHeader()
		brand := "OGH-ZIV"
		adminTG := "-"
		if data, err := os.ReadFile(STRF); err == nil {
			conf := parseConf(string(data))
			if b := conf["BRAND"]; b != "" {
				brand = b
			}
			if t := conf["ADMIN_TG"]; t != "" {
				adminTG = t
			}
		}
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  🛒  MENU JUALAN%s", IT, AL, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  📨  Template Pesan Akun", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[2]%s  📤  Kirim Akun via Telegram", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[3]%s  ⚙️   Pengaturan Toko", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[0]%s  ◀   Kembali", LR, NC))
		boxBot()
		fmt.Println()
		fmt.Printf("  %sBrand: %s%-20s%s  %sTG: @%s%s\n\n", DIM, AL, brand, DIM, DIM, adminTG, NC)

		ch := readLine(fmt.Sprintf("  %s›%s ", A1, NC))
		switch ch {
		case "1":
			tAkun()
		case "2":
			tgKirimAkun()
		case "3":
			setStore()
		case "0":
			return
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

func menuTelegram() {
	for {
		showHeader()
		bstat := LR + "Belum dikonfigurasi" + NC
		if data, err := os.ReadFile(BOTF); err == nil {
			if botName := parseConf(string(data))["BOT_NAME"]; botName != "" {
				bstat = LG + "@" + botName + NC
			}
		}
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  🤖  TELEGRAM BOT%s", IT, AL, NC))
		boxSep()
		fmt.Printf("  %s║%s  %sStatus :%s %s\n", A1, NC, DIM, NC, bstat)
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  🔧  Setup / Konfigurasi Bot", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[2]%s  📡  Cek Status Bot", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[3]%s  📤  Kirim Akun ke Telegram", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[4]%s  📢  Broadcast Pesan", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[5]%s  📖  Panduan Buat Bot", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[0]%s  ◀   Kembali", LR, NC))
		boxBot()
		fmt.Println()

		ch := readLine(fmt.Sprintf("  %s›%s ", A1, NC))
		switch ch {
		case "1":
			tgSetup()
		case "2":
			tgStatus()
		case "3":
			tgKirimAkun()
		case "4":
			tgBroadcast()
		case "5":
			tgGuide()
		case "0":
			return
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

func menuService() {
	for {
		showHeader()
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  ⚙️   MANAJEMEN SERVICE%s", IT, AL, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  🖥️   Status Service", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[2]%s  ▶️   Start ZiVPN", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[3]%s  ⏹️   Stop ZiVPN", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[4]%s  🔄  Restart ZiVPN", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[5]%s  📄  Lihat Log", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[6]%s  🔧  Ganti Port", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[7]%s  🌐  Manajemen Domain", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[8]%s  💾  Backup Data", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[9]%s  ♻️   Restore Data", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[0]%s  ◀   Kembali", LR, NC))
		boxBot()
		fmt.Println()

		ch := readLine(fmt.Sprintf("  %s›%s ", A1, NC))
		switch ch {
		case "1":
			svcStatus()
		case "2":
			exec.Command("systemctl", "start", "zivpn").Run()
			ok("ZiVPN dijalankan.")
			pause()
		case "3":
			exec.Command("systemctl", "stop", "zivpn").Run()
			ok("ZiVPN dihentikan.")
			pause()
		case "4":
			exec.Command("systemctl", "restart", "zivpn").Run()
			time.Sleep(1 * time.Second)
			if isUp() {
				ok("Restart berhasil!")
			} else {
				errMsg("Gagal restart!")
			}
			pause()
		case "5":
			svcLog()
		case "6":
			svcPort()
		case "7":
			menuDomain()
		case "8":
			svcBackup()
		case "9":
			svcRestore()
		case "0":
			return
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

func menuDomain() {
	for {
		showHeader()
		curDomain := getDomain()
		curIP := getIP()
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  🌐  MANAJEMEN DOMAIN%s", IT, AL, NC))
		boxSep()
		fmt.Printf("  %s║%s  %sDomain aktif%s : %s%-36s%s  %s║%s\n", A1, NC, DIM, NC, W, curDomain, NC, A1, NC)
		fmt.Printf("  %s║%s  %sIP Publik   %s : %s%-36s%s  %s║%s\n", A1, NC, DIM, NC, A3, curIP, NC, A1, NC)
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  ✏️   Set / Ganti Domain", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[2]%s  🔄  Gunakan IP Publik (hapus domain)", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[3]%s  🔍  Cek DNS Domain", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[4]%s  🔐  Update SSL untuk Domain Baru", A2, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[0]%s  ◀   Kembali", LR, NC))
		boxBot()
		fmt.Println()

		ch := readLine(fmt.Sprintf("  %s›%s ", A1, NC))
		switch ch {
		case "1":
			domainSet()
		case "2":
			domainUseIP()
		case "3":
			domainCheck()
		case "4":
			domainSSL()
		case "0":
			return
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

// ════════════════════════════════════════════════════════════
//  MENU UTAMA
// ════════════════════════════════════════════════════════════

func mainMenu() {
	for {
		showHeader()
		boxTop()
		boxBtn(fmt.Sprintf("  %s%s  ✦  OGH-ZIV PREMIUM PANEL  ✦%s", IT, AL, NC))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[1]%s  👤  Kelola Akun User", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[2]%s  ⚙️   Manajemen Service", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[3]%s  🤖  Telegram Bot", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[4]%s  🛒  Menu Jualan", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[5]%s  📊  Bandwidth & Koneksi", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[6]%s  🔄  Restart Service", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[7]%s  🚀  Install ZiVPN", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[8]%s  🌐  Manajemen Domain", A2, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[9]%s  🎨  Ganti Tema  [ %s ]", A2, NC, THEME_NAME))
		boxSep()
		boxBtn(fmt.Sprintf("  %s[E]%s  🗑️   Uninstall ZiVPN", LR, NC))
		boxSep0()
		boxBtn(fmt.Sprintf("  %s[0]%s  ❌  Keluar", LR, NC))
		boxBot()
		fmt.Println()

		ch := strings.ToLower(readLine(fmt.Sprintf("  %s›%s ", A1, NC)))
		switch ch {
		case "1":
			menuAkun()
		case "2":
			menuService()
		case "3":
			menuTelegram()
		case "4":
			menuJualan()
		case "5":
			svcBandwidth()
		case "6":
			exec.Command("systemctl", "restart", "zivpn").Run()
			time.Sleep(1 * time.Second)
			if isUp() {
				ok("Service berhasil direstart!")
			} else {
				errMsg("Gagal restart!")
			}
			pause()
		case "7":
			doInstall()
		case "8":
			menuDomain()
		case "9":
			menuTema()
		case "e":
			doUninstall()
		case "0":
			fmt.Printf("\n  %s%sSampai jumpa! — OGH-ZIV Premium%s\n\n", IT, AL, NC)
			os.Exit(0)
		default:
			warn("Pilihan tidak valid!")
			time.Sleep(1 * time.Second)
		}
	}
}

// ════════════════════════════════════════════════════════════
//  MAIN
// ════════════════════════════════════════════════════════════

func main() {
	rand.Seed(time.Now().UnixNano())
	checkOS()
	checkRoot()
	os.MkdirAll(DIR, 0755)
	loadTheme()

	if len(os.Args) > 1 && os.Args[1] == "--check-maxlogin" {
		checkMaxloginAll()
		os.Exit(0)
	}

	setupMenuCmd()
	mainMenu()
}
