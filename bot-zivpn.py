#!/usr/bin/env python3
# ================================================================
#   OGH-ZIV STORE BOT — IMPROVED
#   Pembayaran DANA → konfirmasi admin → akun dibuat otomatis
#   ✅ Sinkron dengan ogh-ziv.sh (chanelog/Socks)
#   ✅ Tampilkan IP server setiap buat akun
#   ✅ Tampilkan akun aktif di panel VPS setelah create
# ================================================================

import os, re, json, time, random, string, logging, subprocess, socket
from datetime import datetime, timedelta
import telebot
from telebot import types

# ── Path database OGH-ZIV ───────────────────────────────────────
DIR       = "/etc/zivpn"
UDB       = f"{DIR}/users.db"
PAKETDB   = f"{DIR}/paket.db"
ORDERDB   = f"{DIR}/orders.db"
BOTCONF   = f"{DIR}/bot.conf"
DANACONF  = f"{DIR}/dana.conf"
ADMINCONF = f"{DIR}/admin.conf"
DOMAINF   = f"{DIR}/domain.conf"
MAXLOGDB  = f"{DIR}/maxlogin.db"
CFGJSON   = f"{DIR}/config.json"
LOGFILE   = f"{DIR}/bot.log"
IPCONF    = f"{DIR}/server.ip"   # ← file cache IP server

# ── Logging ─────────────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    handlers=[logging.FileHandler(LOGFILE), logging.StreamHandler()]
)
log = logging.getLogger(__name__)

# ================================================================
#  BACA CONFIG
# ================================================================
def _rc(path):
    d = {}
    try:
        for line in open(path):
            line = line.strip()
            if "=" in line and not line.startswith("#"):
                k, v = line.split("=", 1)
                d[k.strip()] = v.strip()
    except Exception:
        pass
    return d

_bc = _rc(BOTCONF)
_dc = _rc(DANACONF)
_ac = _rc(ADMINCONF)

BOT_TOKEN   = _bc.get("BOT_TOKEN", "")
ADMIN_ID    = int(_bc.get("CHAT_ID", "0"))
ADMIN_UNAME = _ac.get("ADMIN_USERNAME", "admin")

DANA = {
    "no"  : _dc.get("DANA_NO",   "-"),
    "name": _dc.get("DANA_NAME", "-"),
}

if not BOT_TOKEN:
    raise SystemExit("❌ BOT_TOKEN kosong di /etc/zivpn/bot.conf")

bot = telebot.TeleBot(BOT_TOKEN, parse_mode="Markdown")

# ── State per user ───────────────────────────────────────────────
STATE = {}

# ================================================================
#  HELPER UMUM
# ================================================================
def rp(n):
    try:
        return f"{int(n):,}".replace(",", ".")
    except Exception:
        return str(n)

def gen_oid():
    ts   = str(int(time.time()))[-5:]
    rand = "".join(random.choices(string.digits, k=4))
    return f"ORD{ts}{rand}"

def get_domain():
    try:
        return open(DOMAINF).read().strip()
    except Exception:
        return get_server_ip()

def get_port():
    try:
        cfg = json.load(open(CFGJSON))
        # Coba beberapa key yang mungkin dipakai ogh-ziv.sh
        listen = cfg.get("listen", cfg.get("port", ":5667"))
        if isinstance(listen, int):
            return str(listen)
        return str(listen).lstrip(":")
    except Exception:
        return "5667"

def get_server_ip():
    """Ambil IP publik server — cache ke /etc/zivpn/server.ip"""
    # Cek cache dulu (refresh tiap 1 jam)
    try:
        if os.path.exists(IPCONF):
            mtime = os.path.getmtime(IPCONF)
            if time.time() - mtime < 3600:
                ip = open(IPCONF).read().strip()
                if ip:
                    return ip
    except Exception:
        pass

    ip = None
    # Coba via subprocess curl (lebih reliable di server)
    cmds = [
        ["curl", "-s", "--max-time", "5", "https://api.ipify.org"],
        ["curl", "-s", "--max-time", "5", "https://ifconfig.me"],
        ["curl", "-s", "--max-time", "5", "https://icanhazip.com"],
    ]
    for cmd in cmds:
        try:
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=6)
            candidate = r.stdout.strip()
            if re.match(r"^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$", candidate):
                ip = candidate
                break
        except Exception:
            continue

    # Fallback: socket
    if not ip:
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            s.connect(("8.8.8.8", 80))
            ip = s.getsockname()[0]
            s.close()
        except Exception:
            ip = "0.0.0.0"

    try:
        with open(IPCONF, "w") as f:
            f.write(ip)
    except Exception:
        pass
    return ip

def ks(n):
    return "Unlimited" if n == 0 else f"{n} GB"

def send(uid, text, **kw):
    try:
        bot.send_message(uid, text, parse_mode="Markdown", **kw)
    except Exception as e:
        log.warning(f"send err uid={uid}: {e}")

def to_admin(text, **kw):
    send(ADMIN_ID, text, **kw)

def is_admin(uid):
    return int(uid) == ADMIN_ID

# ================================================================
#  DATABASE PAKET
# ================================================================
PAKET_DEFAULT = [
    {"nama": "UDP 7 Hari",  "hari": 7,  "harga": 3000,  "kuota": 0, "maxlogin": 2},
    {"nama": "UDP 15 Hari", "hari": 15, "harga": 6000,  "kuota": 0, "maxlogin": 2},
    {"nama": "UDP 30 Hari", "hari": 30, "harga": 10000, "kuota": 0, "maxlogin": 2},
]

def paket_load():
    hasil = []
    try:
        for line in open(PAKETDB):
            p = line.strip().split("|")
            if len(p) >= 5:
                hasil.append({
                    "nama": p[0], "hari": int(p[1]),
                    "harga": int(p[2]), "kuota": int(p[3]),
                    "maxlogin": int(p[4]),
                })
    except Exception:
        pass
    return hasil

def paket_save(lst):
    with open(PAKETDB, "w") as f:
        for p in lst:
            f.write(f"{p['nama']}|{p['hari']}|{p['harga']}|{p['kuota']}|{p['maxlogin']}\n")

def paket_init():
    if not os.path.exists(PAKETDB) or os.path.getsize(PAKETDB) == 0:
        paket_save(PAKET_DEFAULT)

# ================================================================
#  DATABASE ORDER
# ================================================================
def order_save(o):
    with open(ORDERDB, "a") as f:
        f.write(
            f"{o['id']}|{o['uid']}|{o['uname']}|{o['paket']}|"
            f"{o['hari']}|{o['harga']}|{o['kuota']}|{o['maxlogin']}|"
            f"{o['user']}|{o['pass']}|{o['status']}|{o['tgl']}\n"
        )

def order_load():
    hasil = []
    try:
        for line in open(ORDERDB):
            p = line.strip().split("|")
            if len(p) >= 12:
                hasil.append({
                    "id": p[0], "uid": int(p[1]),
                    "uname": p[2], "paket": p[3],
                    "hari": int(p[4]), "harga": int(p[5]),
                    "kuota": int(p[6]), "maxlogin": int(p[7]),
                    "user": p[8], "pass": p[9],
                    "status": p[10], "tgl": p[11],
                })
    except Exception:
        pass
    return hasil

def order_get(oid):
    for o in order_load():
        if o["id"] == oid:
            return o
    return None

def order_set_status(oid, status):
    rows = order_load()
    with open(ORDERDB, "w") as f:
        for o in rows:
            if o["id"] == oid:
                o["status"] = status
            f.write(
                f"{o['id']}|{o['uid']}|{o['uname']}|{o['paket']}|"
                f"{o['hari']}|{o['harga']}|{o['kuota']}|{o['maxlogin']}|"
                f"{o['user']}|{o['pass']}|{o['status']}|{o['tgl']}\n"
            )

def order_pending_user(uid):
    for o in reversed(order_load()):
        if o["uid"] == uid and o["status"] == "pending_payment":
            return o
    return None

# ================================================================
#  AKUN VPN — sinkron dengan ogh-ziv.sh
# ================================================================
def akun_exists(username):
    try:
        for line in open(UDB):
            if line.startswith(f"{username}|"):
                return True
    except Exception:
        pass
    return False

def akun_list_aktif():
    """Kembalikan list akun yang masih aktif (belum expired)"""
    hasil = []
    today = datetime.now().strftime("%Y-%m-%d")
    try:
        for line in open(UDB):
            line = line.strip()
            if not line:
                continue
            p = line.split("|")
            if len(p) >= 3 and p[2] >= today:
                hasil.append(p)
    except Exception:
        pass
    return hasil

def akun_create(o):
    """Buat akun & sinkron config.json — sesuai pola ogh-ziv.sh"""
    exp = (datetime.now() + timedelta(days=o["hari"])).strftime("%Y-%m-%d")

    # Tulis ke users.db
    with open(UDB, "a") as f:
        f.write(f"{o['user']}|{o['pass']}|{exp}|{o['kuota']}|Pembeli:@{o['uname']}\n")

    # Update maxlogin.db
    subprocess.run(["sed", "-i", f"/^{o['user']}|/d", MAXLOGDB], capture_output=True)
    with open(MAXLOGDB, "a") as f:
        f.write(f"{o['user']}|{o['maxlogin']}\n")

    vpn_reload()
    return exp

def akun_hapus(username):
    subprocess.run(["sed", "-i", f"/^{username}|/d", UDB],      capture_output=True)
    subprocess.run(["sed", "-i", f"/^{username}|/d", MAXLOGDB], capture_output=True)
    vpn_reload()

def vpn_reload():
    """
    Rebuild config.json auth passwords dari users.db
    Sesuai pola ogh-ziv.sh yang menyimpan password di config.json["auth"]["config"]
    """
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
        pws   = ['"' + l.split("|")[1] + '"' for l in lines if len(l.split("|")) >= 2]
        cfg   = json.load(open(CFGJSON))

        # Patch auth config — sesuai struktur ogh-ziv / zahidbd2
        if "auth" in cfg and isinstance(cfg["auth"], dict):
            cfg["auth"]["config"] = pws
        else:
            # Fallback: simpan langsung sebagai array di root
            cfg["passwords"] = pws

        json.dump(cfg, open(CFGJSON, "w"), indent=2)
        subprocess.run(["systemctl", "restart", "zivpn"], capture_output=True)
        log.info("VPN reloaded OK")
    except Exception as e:
        log.error(f"vpn_reload: {e}")

def dana_save():
    with open(DANACONF, "w") as f:
        f.write(f"DANA_NO={DANA['no']}\nDANA_NAME={DANA['name']}\n")

# ================================================================
#  PANEL VPS — Akun Aktif
# ================================================================
def build_panel_aktif():
    """Bangun teks panel akun aktif untuk ditampilkan di bot"""
    aktif = akun_list_aktif()
    ip    = get_server_ip()
    port  = get_port()
    now   = datetime.now().strftime("%Y-%m-%d %H:%M")

    if not aktif:
        return (
            f"📊 *PANEL VPS — OGH-ZIV*\n"
            f"🖥️ IP Server  : `{ip}`\n"
            f"🔌 Port       : `{port}`\n"
            f"🕐 Update     : {now}\n\n"
            f"_Belum ada akun aktif._"
        )

    lines = [
        f"📊 *PANEL VPS — OGH-ZIV*",
        f"🖥️ IP Server  : `{ip}`",
        f"🔌 Port       : `{port}`",
        f"✅ Akun Aktif : {len(aktif)}",
        f"🕐 Update     : {now}",
        f"",
        f"{'─'*30}",
    ]
    for i, p in enumerate(aktif[:25], 1):
        user = p[0]
        exp  = p[2] if len(p) > 2 else "-"
        # Hitung sisa hari
        try:
            sisa = (datetime.strptime(exp, "%Y-%m-%d") - datetime.now()).days
            sisa_txt = f"{sisa}h" if sisa >= 0 else "expired"
        except Exception:
            sisa_txt = "-"
        lines.append(f"{i:>2}. `{user}` — exp:{exp} ({sisa_txt})")

    if len(aktif) > 25:
        lines.append(f"_...dan {len(aktif)-25} lainnya_")

    return "\n".join(lines)

def kirim_panel_aktif_ke_admin(label=""):
    """Kirim panel VPS ke admin setelah ada akun dibuat/dihapus"""
    header = f"🔄 *{label}*\n\n" if label else ""
    to_admin(header + build_panel_aktif())

# ================================================================
#  TAMPILAN
# ================================================================
EMO_STATUS = {
    "pending_payment": "⏳ Menunggu Bayar",
    "waiting_confirm": "🔍 Verifikasi Admin",
    "done":            "🎉 Aktif",
    "rejected":        "❌ Ditolak",
}

def show_start(uid, fname):
    ip = get_server_ip()
    kb = types.InlineKeyboardMarkup(row_width=2)
    kb.add(
        types.InlineKeyboardButton("🛒 Beli Paket",    callback_data="m_beli"),
        types.InlineKeyboardButton("📋 Cek Order",     callback_data="m_order"),
    )
    kb.add(
        types.InlineKeyboardButton("❓ Cara Beli",     callback_data="m_help"),
        types.InlineKeyboardButton("📞 Hubungi Admin", callback_data="m_admin"),
    )
    send(uid,
        f"👋 Halo *{fname}*!\n\n"
        f"Selamat datang di *🔐 OGH-ZIV UDP VPN Store*\n\n"
        f"╔══════════════════════════\n"
        f"║  📡 VPN UDP Premium\n"
        f"║  ⚡ Cepat & Stabil\n"
        f"║  💰 Mulai Rp 3.000\n"
        f"║  🖥️ Server: `{ip}`\n"
        f"╚══════════════════════════\n\n"
        f"Pilih menu:",
        reply_markup=kb
    )

def show_paket(uid):
    daftar = paket_load()
    if not daftar:
        send(uid, "❌ Belum ada paket. Hubungi admin.")
        return
    text = "📦 *PAKET UDP VPN*\n\n"
    kb   = types.InlineKeyboardMarkup(row_width=1)
    for i, p in enumerate(daftar):
        text += (
            f"┌──────────────────────────\n"
            f"│ 📦 *{p['nama']}*\n"
            f"│ ⏰ Masa    : {p['hari']} hari\n"
            f"│ 💰 Harga  : *Rp {rp(p['harga'])}*\n"
            f"│ 📡 Kuota  : {ks(p['kuota'])}\n"
            f"│ 🔒 Device : {p['maxlogin']} login\n"
            f"└──────────────────────────\n\n"
        )
        kb.add(types.InlineKeyboardButton(
            f"🛒 {p['nama']} — Rp {rp(p['harga'])}",
            callback_data=f"beli_{i}",
        ))
    kb.add(types.InlineKeyboardButton("« Kembali", callback_data="m_back"))
    send(uid, text, reply_markup=kb)

def show_cek_order(uid):
    milik = [o for o in order_load() if o["uid"] == uid]
    if not milik:
        kb = types.InlineKeyboardMarkup()
        kb.add(types.InlineKeyboardButton("🛒 Beli Sekarang", callback_data="m_beli"))
        send(uid,
            "📋 *CEK ORDER*\n\n"
            "Kamu belum punya order.\n",
            reply_markup=kb
        )
        return
    text = "📋 *RIWAYAT ORDER KAMU*\n\n"
    for o in milik[-5:]:
        st = EMO_STATUS.get(o["status"], o["status"])
        text += (
            f"🆔 `{o['id']}`\n"
            f"📦 {o['paket']} | 💰 Rp {rp(o['harga'])}\n"
            f"👤 Akun : `{o['user']}`\n"
            f"📊 Status: *{st}*\n"
            f"📅 {o['tgl']}\n\n"
        )
    kb = types.InlineKeyboardMarkup()
    kb.add(types.InlineKeyboardButton("« Kembali", callback_data="m_back"))
    send(uid, text, reply_markup=kb)

def show_help(uid):
    kb = types.InlineKeyboardMarkup(row_width=1)
    kb.add(types.InlineKeyboardButton("🛒 Beli Sekarang",  callback_data="m_beli"))
    kb.add(types.InlineKeyboardButton("📞 Hubungi Admin",  url=f"https://t.me/{ADMIN_UNAME}"))
    kb.add(types.InlineKeyboardButton("« Kembali",         callback_data="m_back"))
    send(uid,
        "❓ *CARA BELI VPN*\n\n"
        "1️⃣ Tekan *🛒 Beli Paket* → pilih paket\n"
        "2️⃣ Masukkan *username* & *password* VPN\n"
        "3️⃣ *Transfer DANA* sesuai nominal\n"
        "4️⃣ Kirim *📸 screenshot bukti transfer* ke bot\n"
        "5️⃣ Admin verifikasi → akun langsung aktif ✅\n\n"
        "⏱ Konfirmasi biasanya *< 5 menit*\n\n"
        "📋 *Harga Paket:*\n"
        "• UDP 7 Hari  → Rp 3.000\n"
        "• UDP 15 Hari → Rp 6.000\n"
        "• UDP 30 Hari → Rp 10.000\n\n"
        f"📞 Keluhan / konfirmasi:\n➡️ @{ADMIN_UNAME}",
        reply_markup=kb
    )

def show_admin_contact(uid):
    kb = types.InlineKeyboardMarkup()
    kb.add(types.InlineKeyboardButton(
        f"💬 Chat Admin @{ADMIN_UNAME}",
        url=f"https://t.me/{ADMIN_UNAME}"
    ))
    kb.add(types.InlineKeyboardButton("« Kembali", callback_data="m_back"))
    send(uid,
        f"📞 *HUBUNGI ADMIN*\n\n"
        f"👤 Admin : @{ADMIN_UNAME}\n"
        f"🕐 Respon : Secepatnya\n\n"
        f"Untuk:\n"
        f"• Konfirmasi pembayaran\n"
        f"• Laporan masalah VPN\n"
        f"• Pertanyaan seputar layanan\n\n"
        f"_Sertakan Order ID saat menghubungi_",
        reply_markup=kb
    )

def notif_order_ke_admin(o):
    kb = types.InlineKeyboardMarkup(row_width=2)
    kb.add(
        types.InlineKeyboardButton("✅ Konfirmasi", callback_data=f"adm_ok_{o['id']}"),
        types.InlineKeyboardButton("❌ Tolak",      callback_data=f"adm_no_{o['id']}"),
    )
    to_admin(
        f"📸 *BUKTI BAYAR MASUK*\n\n"
        f"🆔 Order  : `{o['id']}`\n"
        f"👤 Pembeli: @{o['uname']} (`{o['uid']}`)\n"
        f"📦 Paket  : {o['paket']}\n"
        f"💰 Harga  : Rp {rp(o['harga'])}\n\n"
        f"👤 Username VPN : `{o['user']}`\n"
        f"🔑 Password VPN : `{o['pass']}`\n\n"
        f"Tekan tombol atau ketik:\n"
        f"✅ `/confirm {o['id']}`\n"
        f"❌ `/reject {o['id']}`",
        reply_markup=kb
    )

def kirim_akun_ke_user(o, exp):
    """Kirim detail akun ke user — tampilkan IP server"""
    ip   = get_server_ip()
    host = get_domain()
    port = get_port()

    kb = types.InlineKeyboardMarkup()
    kb.add(types.InlineKeyboardButton(
        f"📞 Admin @{ADMIN_UNAME}",
        url=f"https://t.me/{ADMIN_UNAME}"
    ))
    send(o["uid"],
        f"🎉 *PEMBAYARAN DIKONFIRMASI!*\n\n"
        f"Akun VPN kamu sudah *AKTIF* ✅\n\n"
        f"╔═══════════════════════════\n"
        f"║ 👤 Username : `{o['user']}`\n"
        f"║ 🔑 Password : `{o['pass']}`\n"
        f"╠═══════════════════════════\n"
        f"║ 🌐 Host/Domain : `{host}`\n"
        f"║ 🖥️ IP Server   : `{ip}`\n"
        f"║ 🔌 Port        : `{port}`\n"
        f"║ 📡 Protocol    : `ZiVPN UDP`\n"
        f"╠═══════════════════════════\n"
        f"║ 📦 Kuota    : {ks(o['kuota'])}\n"
        f"║ 🔒 MaxLogin : {o['maxlogin']} device\n"
        f"║ 📅 Expired  : {exp}\n"
        f"╚═══════════════════════════\n\n"
        f"📱 *Download ZiVPN* di Play Store / App Store\n"
        f"⚠️ Jangan share akun ke orang lain!\n\n"
        f"📞 Ada masalah? Hubungi @{ADMIN_UNAME}",
        reply_markup=kb
    )

# ================================================================
#  HANDLER FOTO (bukti transfer)
# ================================================================
@bot.message_handler(content_types=["photo"])
def on_photo(msg):
    uid = msg.from_user.id
    pending = order_pending_user(uid)
    if not pending:
        send(uid,
            "❓ *Tidak ada order yang menunggu pembayaran.*\n\n"
            "Ketik /beli untuk membeli paket terlebih dahulu."
        )
        return
    order_set_status(pending["id"], "waiting_confirm")
    try:
        bot.forward_message(ADMIN_ID, uid, msg.message_id)
    except Exception:
        pass
    notif_order_ke_admin(pending)
    kb = types.InlineKeyboardMarkup()
    kb.add(types.InlineKeyboardButton(
        f"📞 Admin @{ADMIN_UNAME}", url=f"https://t.me/{ADMIN_UNAME}"
    ))
    send(uid,
        f"✅ *Bukti pembayaran diterima!*\n\n"
        f"🆔 Order : `{pending['id']}`\n"
        f"📦 Paket : {pending['paket']}\n\n"
        f"Admin sedang memverifikasi.\n"
        f"⏱ Estimasi *< 5 menit*\n\n"
        f"Butuh bantuan? @{ADMIN_UNAME}",
        reply_markup=kb
    )

# ================================================================
#  HANDLER PESAN
# ================================================================
@bot.message_handler(func=lambda m: True)
def on_message(msg):
    uid  = msg.from_user.id
    text = (msg.text or "").strip()

    if uid in STATE:
        handle_state(msg, text)
        return

    cmd = text.split()[0].lower() if text else ""

    if cmd in ("/start", "/menu"):
        show_start(uid, msg.from_user.first_name)
    elif cmd == "/beli":
        show_paket(uid)
    elif cmd == "/cekorder":
        show_cek_order(uid)
    elif cmd == "/help":
        show_help(uid)
    elif cmd == "/admin":
        show_admin_contact(uid)

    # ── Panel VPS (siapa pun bisa /panel untuk cek status) ───────
    elif cmd == "/panel":
        if is_admin(uid):
            send(uid, build_panel_aktif())
        else:
            send(uid, "⛔ Perintah ini hanya untuk admin.")

    # ── Admin commands ───────────────────────────────────────────
    elif is_admin(uid) and text.startswith("/confirm "):
        cmd_confirm(uid, text[9:].strip())
    elif is_admin(uid) and text.startswith("/reject "):
        cmd_reject(uid, text[8:].strip())
    elif is_admin(uid) and cmd == "/listorder":
        cmd_listorder(uid)
    elif is_admin(uid) and cmd == "/listakun":
        cmd_listakun(uid)
    elif is_admin(uid) and text.startswith("/hapusakun "):
        u = text[11:].strip()
        akun_hapus(u)
        send(uid, f"✅ Akun `{u}` dihapus & VPN direload.")
        kirim_panel_aktif_ke_admin(f"Akun `{u}` dihapus")
    elif is_admin(uid) and cmd == "/listpaket":
        cmd_listpaket(uid)
    elif is_admin(uid) and cmd == "/newpaket":
        STATE[uid] = {"step": "np_nama"}
        send(uid, "📦 Masukkan *nama paket*:\nContoh: `UDP 30 Hari`")
    elif is_admin(uid) and text.startswith("/delpaket "):
        try:
            cmd_delpaket(uid, int(text[10:].strip()) - 1)
        except Exception:
            send(uid, "❌ Format: `/delpaket 1`")
    elif is_admin(uid) and cmd == "/setdana":
        STATE[uid] = {"step": "set_dana_no"}
        send(uid, f"💳 No DANA saat ini: `{DANA['no']}`\n\nMasukkan *nomor DANA baru*:")
    elif is_admin(uid) and cmd == "/broadcast":
        STATE[uid] = {"step": "broadcast"}
        send(uid, "📢 Ketik pesan broadcast:")
    elif is_admin(uid) and cmd == "/info":
        cmd_info(uid)
    elif is_admin(uid) and cmd == "/buatakun":
        show_paket_admin_free(uid)
    elif is_admin(uid) and cmd == "/ip":
        ip = get_server_ip()
        send(uid, f"🖥️ *IP Server:* `{ip}`\n🔌 *Port:* `{get_port()}`")
    else:
        show_start(uid, msg.from_user.first_name)


# ================================================================
#  CALLBACK
# ================================================================
@bot.callback_query_handler(func=lambda c: True)
def on_callback(call):
    uid  = call.from_user.id
    data = call.data
    bot.answer_callback_query(call.id)

    if data == "m_beli":
        show_paket(uid)
    elif data == "m_order":
        show_cek_order(uid)
    elif data == "m_help":
        show_help(uid)
    elif data == "m_admin":
        show_admin_contact(uid)
    elif data == "m_back":
        show_start(uid, call.from_user.first_name)

    elif data.startswith("beli_"):
        idx    = int(data[5:])
        daftar = paket_load()
        if idx < 0 or idx >= len(daftar):
            return
        p = daftar[idx]
        STATE[uid] = {"step": "input_username", "pidx": idx}
        send(uid,
            f"✅ *{p['nama']}* dipilih\n\n"
            f"⏰ {p['hari']} hari | 💰 Rp {rp(p['harga'])} | "
            f"📡 {ks(p['kuota'])} | 🔒 {p['maxlogin']} device\n\n"
            f"👤 Masukkan *username* untuk akun VPN:\n"
            f"_(3-20 karakter: huruf, angka, underscore)_"
        )

    elif data.startswith("bukti_"):
        oid = data[6:]
        o = order_get(oid)
        if not o or o["status"] != "pending_payment":
            send(uid, "❓ Order tidak ditemukan atau sudah diproses.")
            return
        order_set_status(oid, "waiting_confirm")
        notif_order_ke_admin(o)
        send(uid,
            f"✅ *Admin sudah diberitahu!*\n\n"
            f"🆔 Order : `{oid}`\n"
            f"⏱ Estimasi konfirmasi *< 5 menit*\n\n"
            f"Jika lebih dari 5 menit, hubungi @{ADMIN_UNAME}"
        )

    elif data.startswith("adm_ok_") and is_admin(uid):
        cmd_confirm(uid, data[7:])

    elif data.startswith("adm_no_") and is_admin(uid):
        cmd_reject(uid, data[7:])

    elif data.startswith("adm_free_") and is_admin(uid):
        idx    = int(data[9:])
        daftar = paket_load()
        if idx < 0 or idx >= len(daftar):
            return
        p = daftar[idx]
        STATE[uid] = {
            "step"    : "adm_free_user",
            "kuota"   : p["kuota"],
            "maxlogin": p["maxlogin"],
            "hari"    : p["hari"],
        }
        send(uid, f"📦 Paket: *{p['nama']}* ({p['hari']} hari)\n\n👤 Masukkan *username* akun:")

# ================================================================
#  ADMIN COMMANDS
# ================================================================
def show_paket_admin_free(uid):
    daftar = paket_load()
    if not daftar:
        STATE[uid] = {"step": "adm_free_user", "kuota": 0, "maxlogin": 2, "hari": 0}
        send(uid, "👤 Masukkan *username* akun gratis:")
        return
    text = "📦 *Pilih paket untuk akun GRATIS:*\n\n"
    kb   = types.InlineKeyboardMarkup(row_width=1)
    for i, p in enumerate(daftar):
        text += f"• {p['nama']} ({p['hari']} hari)\n"
        kb.add(types.InlineKeyboardButton(
            f"📦 {p['nama']} ({p['hari']} hari)",
            callback_data=f"adm_free_{i}"
        ))
    send(uid, text, reply_markup=kb)

def cmd_confirm(uid, oid):
    o = order_get(oid)
    if not o:
        send(uid, f"❌ Order tidak ditemukan: `{oid}`"); return
    if o["status"] == "done":
        send(uid, "⚠️ Order ini sudah selesai."); return
    exp = akun_create(o)
    order_set_status(oid, "done")
    kirim_akun_ke_user(o, exp)
    ip = get_server_ip()
    send(uid,
        f"✅ *Order `{oid}` dikonfirmasi!*\n\n"
        f"👤 Akun `{o['user']}` aktif hingga *{exp}*\n"
        f"🖥️ IP Server: `{ip}`"
    )
    log.info(f"confirmed order={oid} akun={o['user']} exp={exp}")
    # Kirim panel VPS terbaru ke admin
    kirim_panel_aktif_ke_admin(f"Akun baru: {o['user']} (exp:{exp})")

def cmd_reject(uid, oid):
    o = order_get(oid)
    if not o:
        send(uid, f"❌ Order tidak ditemukan: `{oid}`"); return
    order_set_status(oid, "rejected")
    kb = types.InlineKeyboardMarkup()
    kb.add(types.InlineKeyboardButton(
        f"📞 Admin @{ADMIN_UNAME}", url=f"https://t.me/{ADMIN_UNAME}"
    ))
    send(o["uid"],
        f"❌ *Order `{oid}` ditolak.*\n\n"
        f"Kemungkinan:\n"
        f"• Bukti pembayaran tidak jelas\n"
        f"• Nominal tidak sesuai\n\n"
        f"Hubungi admin untuk info lebih lanjut:",
        reply_markup=kb
    )
    send(uid, f"✅ Order `{oid}` ditolak.")

def cmd_listorder(uid):
    rows = order_load()
    if not rows:
        send(uid, "📋 Belum ada order."); return
    E = {"pending_payment": "⏳", "waiting_confirm": "🔍", "done": "🎉", "rejected": "❌"}
    text = "📋 *LIST ORDER (10 terbaru)*\n\n"
    for o in rows[-10:]:
        e = E.get(o["status"], "❓")
        text += (
            f"{e} `{o['id']}`\n"
            f"   @{o['uname']} | {o['paket']} | Rp {rp(o['harga'])}\n"
            f"   👤 `{o['user']}` | {o['tgl']}\n\n"
        )
    text += "✅ `/confirm ID`   ❌ `/reject ID`"
    send(uid, text)

def cmd_listakun(uid):
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
    except Exception:
        lines = []
    if not lines:
        send(uid, "👤 Belum ada akun."); return
    today   = datetime.now().strftime("%Y-%m-%d")
    aktif   = sum(1 for l in lines if len(l.split("|")) >= 3 and l.split("|")[2] >= today)
    expired = len(lines) - aktif
    ip      = get_server_ip()
    text = (
        f"👤 *LIST AKUN VPN*\n"
        f"🖥️ IP Server: `{ip}`\n"
        f"Total: {len(lines)} | ✅ Aktif: {aktif} | ❌ Expired: {expired}\n\n"
    )
    for line in lines[:30]:
        p = line.split("|")
        if len(p) < 3: continue
        st = "✅" if p[2] >= today else "❌"
        text += f"{st} `{p[0]}` exp:{p[2]}\n"
    if len(lines) > 30:
        text += f"_...dan {len(lines)-30} lainnya_"
    send(uid, text)

def cmd_listpaket(uid):
    daftar = paket_load()
    if not daftar:
        send(uid, "📦 Belum ada paket."); return
    text = "📦 *LIST PAKET*\n\n"
    for i, p in enumerate(daftar):
        text += (
            f"*{i+1}. {p['nama']}*\n"
            f"   ⏰ {p['hari']} hari | 💰 Rp {rp(p['harga'])} | "
            f"📡 {ks(p['kuota'])} | 🔒 {p['maxlogin']} device\n"
            f"   Hapus: `/delpaket {i+1}`\n\n"
        )
    send(uid, text)

def cmd_delpaket(uid, idx):
    daftar = paket_load()
    if idx < 0 or idx >= len(daftar):
        send(uid, "❌ Nomor paket tidak valid."); return
    nama = daftar[idx]["nama"]
    daftar.pop(idx)
    paket_save(daftar)
    send(uid, f"✅ Paket *{nama}* dihapus.")

def cmd_info(uid):
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
    except Exception:
        lines = []
    today   = datetime.now().strftime("%Y-%m-%d")
    expired = sum(1 for l in lines if len(l.split("|")) >= 3 and l.split("|")[2] < today)
    rows    = order_load()
    pending = sum(1 for o in rows if o["status"] in ("pending_payment", "waiting_confirm"))
    done    = sum(1 for o in rows if o["status"] == "done")
    omzet   = sum(o["harga"] for o in rows if o["status"] == "done")
    ip      = get_server_ip()
    send(uid,
        f"📊 *INFO PANEL OGH-ZIV*\n\n"
        f"🖥️ IP Server   : `{ip}`\n"
        f"🌐 Domain      : `{get_domain()}`\n"
        f"🔌 Port        : `{get_port()}`\n\n"
        f"👤 Total Akun  : {len(lines)}\n"
        f"✅ Aktif       : {len(lines) - expired}\n"
        f"❌ Expired     : {expired}\n\n"
        f"📋 Total Order : {len(rows)}\n"
        f"🎉 Selesai     : {done}\n"
        f"⏳ Pending     : {pending}\n"
        f"💰 Total Omzet : Rp {rp(omzet)}\n\n"
        f"💳 DANA  : `{DANA['no']}` ({DANA['name']})\n"
        f"👤 Admin : @{ADMIN_UNAME}"
    )

# ================================================================
#  STATE MACHINE
# ================================================================
def handle_state(msg, text):
    uid   = msg.from_user.id
    state = STATE[uid]
    step  = state["step"]

    # ─── Alur beli (user bayar dulu) ────────────────────────────
    if step == "input_username":
        if not re.match(r"^[a-zA-Z0-9_]{3,20}$", text):
            send(uid,
                "❌ *Username tidak valid!*\n\n"
                "• 3-20 karakter\n"
                "• Huruf, angka, underscore (_)\n"
                "• Tanpa spasi / simbol"
            )
            return
        if akun_exists(text):
            send(uid, "❌ Username *sudah dipakai*, coba nama lain.")
            return
        state["vuser"] = text
        state["step"]  = "input_password"
        STATE[uid] = state
        send(uid, f"✅ Username: `{text}`\n\n🔑 Masukkan *password* (min 6 karakter):")

    elif step == "input_password":
        if len(text) < 6:
            send(uid, "❌ Password minimal *6 karakter*!")
            return
        state["vpass"] = text
        STATE.pop(uid, None)

        daftar = paket_load()
        p      = daftar[state["pidx"]]
        oid    = gen_oid()
        uname  = msg.from_user.username or str(uid)

        o = {
            "id": oid, "uid": uid, "uname": uname,
            "paket": p["nama"], "hari": p["hari"],
            "harga": p["harga"], "kuota": p["kuota"],
            "maxlogin": p["maxlogin"],
            "user": state["vuser"], "pass": state["vpass"],
            "status": "pending_payment",
            "tgl": datetime.now().strftime("%Y-%m-%d %H:%M"),
        }
        order_save(o)

        kb = types.InlineKeyboardMarkup(row_width=1)
        kb.add(types.InlineKeyboardButton(
            "📸 Sudah Transfer, Beritahu Admin",
            callback_data=f"bukti_{oid}"
        ))
        kb.add(types.InlineKeyboardButton(
            f"📞 Admin @{ADMIN_UNAME}",
            url=f"https://t.me/{ADMIN_UNAME}"
        ))

        send(uid,
            f"✅ *ORDER BERHASIL DIBUAT!*\n\n"
            f"🆔 Order ID : `{oid}`\n"
            f"📦 Paket    : *{p['nama']}*\n"
            f"📅 Masa     : {p['hari']} hari\n\n"
            f"👤 Username : `{state['vuser']}`\n"
            f"🔑 Password : `{state['vpass']}`\n"
            f"⚠️ _Simpan data ini baik-baik!_\n\n"
            f"━━━━━━━━━━━━━━━━━━━━━━━\n"
            f"💳 *TRANSFER DANA SEKARANG*\n"
            f"━━━━━━━━━━━━━━━━━━━━━━━\n"
            f"📱 No DANA : `{DANA['no']}`\n"
            f"👤 A/N     : *{DANA['name']}*\n"
            f"💰 Nominal : *Rp {rp(p['harga'])}*\n"
            f"━━━━━━━━━━━━━━━━━━━━━━━\n\n"
            f"📸 Setelah transfer:\n"
            f"*Kirim screenshot bukti bayar ke chat ini*\n"
            f"atau tekan tombol di bawah\n\n"
            f"⏰ Order batal otomatis dalam *30 menit*",
            reply_markup=kb
        )
        to_admin(
            f"📝 *ORDER MASUK — BELUM BAYAR*\n\n"
            f"🆔 Order  : `{oid}`\n"
            f"👤 Pembeli: @{uname} (`{uid}`)\n"
            f"📦 Paket  : {p['nama']} | Rp {rp(p['harga'])}\n"
            f"👤 Akun   : `{state['vuser']}`\n\n"
            f"⏳ Menunggu bukti pembayaran..."
        )

    # ─── Admin buat akun gratis ──────────────────────────────────
    elif step == "adm_free_user":
        if not re.match(r"^[a-zA-Z0-9_]{3,20}$", text):
            send(uid, "❌ Username tidak valid! (3-20 karakter, huruf/angka/underscore)")
            return
        if akun_exists(text):
            send(uid, "❌ Username sudah ada, coba yang lain.")
            return
        state["vuser"] = text
        state["step"]  = "adm_free_pass"
        STATE[uid] = state
        send(uid, f"✅ Username: `{text}`\n\n🔑 Masukkan *password*:")

    elif step == "adm_free_pass":
        if len(text) < 4:
            send(uid, "❌ Password minimal 4 karakter!")
            return
        state["vpass"] = text
        # Jika hari sudah ada dari pilihan paket, skip tanya hari
        if state.get("hari", 0) > 0:
            STATE.pop(uid, None)
            _buat_akun_free(uid, state, state["hari"])
        else:
            state["step"]  = "adm_free_hari"
            STATE[uid] = state
            send(uid, f"✅ Pass: `{text}`\n\n⏰ Masa aktif *(hari)*:")

    elif step == "adm_free_hari":
        if not text.isdigit() or int(text) < 1:
            send(uid, "❌ Masukkan angka hari!")
            return
        STATE.pop(uid, None)
        _buat_akun_free(uid, state, int(text))

    # ─── Set DANA ─────────────────────────────────────────────────
    elif step == "set_dana_no":
        state["new_no"] = text
        state["step"]   = "set_dana_name"
        STATE[uid] = state
        send(uid, f"✅ No DANA: `{text}`\n\n👤 Masukkan *nama pemilik* (A/N):")

    elif step == "set_dana_name":
        DANA["no"]   = state["new_no"]
        DANA["name"] = text
        dana_save()
        STATE.pop(uid, None)
        send(uid,
            f"✅ *DANA diperbarui!*\n\n"
            f"💳 No  : `{DANA['no']}`\n"
            f"👤 A/N : *{DANA['name']}*"
        )

    # ─── Tambah paket ─────────────────────────────────────────────
    elif step == "np_nama":
        state["np_nama"] = text
        state["step"]    = "np_hari"
        STATE[uid] = state
        send(uid, f"📦 *{text}*\n\n⏰ Masa aktif *(hari)*:")

    elif step == "np_hari":
        if not text.isdigit() or int(text) < 1:
            send(uid, "❌ Masukkan angka hari!"); return
        state["np_hari"] = int(text)
        state["step"]    = "np_harga"
        STATE[uid] = state
        send(uid, f"⏰ {text} hari\n\n💰 Harga *(Rupiah)*:")

    elif step == "np_harga":
        if not text.isdigit():
            send(uid, "❌ Masukkan angka harga!"); return
        state["np_harga"] = int(text)
        state["step"]     = "np_kuota"
        STATE[uid] = state
        send(uid, f"💰 Rp {rp(text)}\n\n📦 Kuota GB *(0=unlimited)*:")

    elif step == "np_kuota":
        if not text.isdigit():
            send(uid, "❌ Masukkan angka!"); return
        state["np_kuota"] = int(text)
        state["step"]     = "np_maxlogin"
        STATE[uid] = state
        send(uid, f"📦 {ks(int(text))}\n\n🔒 Max login device:")

    elif step == "np_maxlogin":
        if not text.isdigit() or int(text) < 1:
            send(uid, "❌ Masukkan angka!"); return
        daftar = paket_load()
        daftar.append({
            "nama": state["np_nama"], "hari": state["np_hari"],
            "harga": state["np_harga"], "kuota": state["np_kuota"],
            "maxlogin": int(text),
        })
        paket_save(daftar)
        STATE.pop(uid, None)
        send(uid,
            f"✅ *Paket ditambahkan!*\n\n"
            f"📦 {state['np_nama']} | ⏰ {state['np_hari']} hari | "
            f"💰 Rp {rp(state['np_harga'])} | 📡 {ks(state['np_kuota'])} | 🔒 {text} device"
        )

    # ─── Broadcast ────────────────────────────────────────────────
    elif step == "broadcast":
        STATE.pop(uid, None)
        to_admin(f"📢 *BROADCAST*\n\n{text}")
        send(uid, "✅ Broadcast dikirim!")


def _buat_akun_free(uid, state, hari):
    """Helper buat akun gratis admin — tampilkan IP & panel aktif"""
    o = {
        "id": gen_oid(), "uid": uid, "uname": "admin",
        "paket": f"FREE {hari} Hari", "hari": hari,
        "harga": 0, "kuota": state.get("kuota", 0),
        "maxlogin": state.get("maxlogin", 2),
        "user": state["vuser"], "pass": state["vpass"],
        "status": "done",
        "tgl": datetime.now().strftime("%Y-%m-%d %H:%M"),
    }
    order_save(o)
    exp  = akun_create(o)
    ip   = get_server_ip()
    host = get_domain()
    port = get_port()

    send(uid,
        f"✅ *AKUN GRATIS BERHASIL DIBUAT*\n\n"
        f"╔═══════════════════════════\n"
        f"║ 👤 Username : `{o['user']}`\n"
        f"║ 🔑 Password : `{o['pass']}`\n"
        f"╠═══════════════════════════\n"
        f"║ 🌐 Host/Domain : `{host}`\n"
        f"║ 🖥️ IP Server   : `{ip}`\n"
        f"║ 🔌 Port        : `{port}`\n"
        f"║ 📡 Protocol    : ZiVPN UDP\n"
        f"╠═══════════════════════════\n"
        f"║ ⏰ Masa Aktif : {hari} hari\n"
        f"║ 📅 Expired   : {exp}\n"
        f"╚═══════════════════════════"
    )
    log.info(f"Admin free akun={o['user']} exp={exp} ip={ip}")

    # Tampilkan panel VPS akun aktif setelah create
    kirim_panel_aktif_ke_admin(f"Akun baru dibuat: {o['user']} (exp:{exp})")


# ================================================================
#  MAIN
# ================================================================
if __name__ == "__main__":
    os.makedirs(DIR, exist_ok=True)
    paket_init()
    ip = get_server_ip()
    log.info(f"OGH-ZIV Bot start | admin={ADMIN_ID} | @{ADMIN_UNAME} | dana={DANA['no']} | ip={ip}")
    print(f"✅ OGH-ZIV Bot berjalan | Admin: @{ADMIN_UNAME} | Server IP: {ip}")
    bot.infinity_polling(timeout=30, long_polling_timeout=30)
