#!/usr/bin/env python3
# ============================================================
#   OGH-ZIV PREMIUM — Telegram Bot Auto Create Akun
#   Terintegrasi dengan OGH-ZIV Panel (ogh-ziv.sh)
#   Pembayaran via DANA — Cek Screenshot Otomatis
#   GitHub: https://github.com/chanelog/Cek-bot
# ============================================================

import os
import re
import json
import logging
import subprocess
import random
import string
from datetime import datetime, timedelta
from pathlib import Path

# ── Telegram Bot Library ─────────────────────────────────────
try:
    from telegram import (
        Update, InlineKeyboardButton, InlineKeyboardMarkup,
        ReplyKeyboardMarkup, KeyboardButton
    )
    from telegram.ext import (
        ApplicationBuilder, CommandHandler, MessageHandler,
        CallbackQueryHandler, ContextTypes, filters,
        ConversationHandler
    )
except ImportError:
    print("Install dulu: pip3 install python-telegram-bot --break-system-packages")
    exit(1)

# ── OCR Library untuk cek screenshot ─────────────────────────
try:
    from PIL import Image
    import pytesseract
    OCR_AVAILABLE = True
except ImportError:
    OCR_AVAILABLE = False
    print("[WARN] pytesseract/Pillow tidak tersedia. OCR tidak aktif.")

# ============================================================
#  KONFIGURASI — Edit bagian ini sesuai kebutuhan
# ============================================================
CONFIG_FILE = "/etc/zivpn/bot_store.conf"
USERS_DB    = "/etc/zivpn/users.db"
DOMAIN_CONF = "/etc/zivpn/domain.conf"
BOT_CONF    = "/etc/zivpn/bot.conf"
MLDB        = "/etc/zivpn/maxlogin.db"

# Paket berbayar
PAKET = {
    "1": {"nama": "3 Hari",  "hari": 3,  "harga": 3000,  "kuota": 0, "maxlogin": 2},
    "2": {"nama": "15 Hari", "hari": 15, "harga": 6000,  "kuota": 0, "maxlogin": 2},
    "3": {"nama": "30 Hari", "hari": 30, "harga": 10000, "kuota": 0, "maxlogin": 2},
}

# Paket trial gratis (120 menit)
TRIAL_MENIT = 120

# State untuk ConversationHandler
(
    ST_MENU, ST_PILIH_PAKET, ST_KIRIM_SS, ST_VERIF,
    ST_ADMIN_MENU, ST_ADMIN_ADD, ST_ADMIN_DEL
) = range(7)

# ── Logger ───────────────────────────────────────────────────
logging.basicConfig(
    format="%(asctime)s [%(levelname)s] %(message)s",
    level=logging.INFO
)
log = logging.getLogger(__name__)

# ============================================================
#  LOAD KONFIGURASI
# ============================================================
def load_config() -> dict:
    cfg = {
        "BOT_TOKEN":   "",
        "ADMIN_IDS":   [],
        "DANA_NUMBER": "08xxxxxxxxxx",
        "DANA_NAME":   "Nama Pemilik",
        "BRAND":       "OGH-ZIV",
        "ADMIN_TG":    "@admin",
    }
    # Load dari /etc/zivpn/bot_store.conf
    if Path(CONFIG_FILE).exists():
        for line in Path(CONFIG_FILE).read_text().splitlines():
            if "=" in line and not line.strip().startswith("#"):
                k, _, v = line.partition("=")
                k = k.strip(); v = v.strip().strip('"').strip("'")
                if k == "BOT_TOKEN":  cfg["BOT_TOKEN"]   = v
                if k == "ADMIN_IDS":
                    try: cfg["ADMIN_IDS"] = [int(x) for x in v.split(",") if x.strip()]
                    except: pass
                if k == "DANA_NUMBER": cfg["DANA_NUMBER"] = v
                if k == "DANA_NAME":   cfg["DANA_NAME"]   = v
                if k == "BRAND":       cfg["BRAND"]       = v
                if k == "ADMIN_TG":    cfg["ADMIN_TG"]    = v

    # Fallback: load token dari bot.conf jika tidak di bot_store.conf
    if not cfg["BOT_TOKEN"] and Path(BOT_CONF).exists():
        for line in Path(BOT_CONF).read_text().splitlines():
            if line.startswith("BOT_TOKEN="):
                cfg["BOT_TOKEN"] = line.split("=", 1)[1].strip()
                break
    return cfg

CFG = load_config()

# ============================================================
#  HELPERS — Panel OGH-ZIV
# ============================================================
def get_ip() -> str:
    try:
        r = subprocess.check_output(
            ["curl", "-s4", "--max-time", "5", "ifconfig.me"],
            stderr=subprocess.DEVNULL
        ).decode().strip()
        if re.match(r"^\d+\.\d+\.\d+\.\d+$", r):
            return r
    except: pass
    try:
        return subprocess.check_output(
            ["hostname", "-I"], stderr=subprocess.DEVNULL
        ).decode().split()[0]
    except:
        return "0.0.0.0"

def get_domain() -> str:
    if Path(DOMAIN_CONF).exists():
        return Path(DOMAIN_CONF).read_text().strip()
    return get_ip()

def get_port() -> str:
    cfg_file = "/etc/zivpn/config.json"
    if Path(cfg_file).exists():
        try:
            data = json.loads(Path(cfg_file).read_text())
            listen = data.get("listen", ":5667")
            return listen.lstrip(":")
        except: pass
    return "5667"

def rand_pass(length: int = 12) -> str:
    chars = string.ascii_letters + string.digits
    return "".join(random.choices(chars, k=length))

def rand_user(prefix: str = "ziv") -> str:
    suffix = "".join(random.choices(string.digits, k=5))
    return f"{prefix}{suffix}"

def user_exists(username: str) -> bool:
    if not Path(USERS_DB).exists():
        return False
    for line in Path(USERS_DB).read_text().splitlines():
        if line.startswith(f"{username}|"):
            return True
    return False

def create_account(username: str, password: str, days: int, kuota: int,
                   maxlogin: int, note: str = "-") -> dict:
    """Buat akun baru di users.db dan reload ZiVPN"""
    exp = (datetime.now() + timedelta(days=days)).strftime("%Y-%m-%d")

    # Append ke users.db
    Path(USERS_DB).parent.mkdir(parents=True, exist_ok=True)
    with open(USERS_DB, "a") as f:
        f.write(f"{username}|{password}|{exp}|{kuota}|{note}\n")

    # Set maxlogin
    mldb = Path(MLDB)
    mldb.parent.mkdir(parents=True, exist_ok=True)
    lines = mldb.read_text().splitlines() if mldb.exists() else []
    lines = [l for l in lines if not l.startswith(f"{username}|")]
    lines.append(f"{username}|{maxlogin}")
    mldb.write_text("\n".join(lines) + "\n")

    # Reload password ke config.json & restart ZiVPN
    _reload_pw()

    return {
        "username": username,
        "password": password,
        "exp":      exp,
        "ip":       get_ip(),
        "domain":   get_domain(),
        "port":     get_port(),
        "kuota":    "Unlimited" if kuota == 0 else f"{kuota} GB",
        "maxlogin": maxlogin,
        "note":     note,
    }

def _reload_pw():
    """Reload password config.json & restart ZiVPN"""
    cfg_file = "/etc/zivpn/config.json"
    if not Path(USERS_DB).exists() or not Path(cfg_file).exists():
        return
    try:
        pws = []
        for line in Path(USERS_DB).read_text().splitlines():
            parts = line.split("|")
            if len(parts) >= 2:
                pws.append(f'"{parts[1]}"')
        data = json.loads(Path(cfg_file).read_text())
        data["auth"]["config"] = json.loads(f"[{','.join(pws)}]")
        Path(cfg_file).write_text(json.dumps(data, indent=2))
        subprocess.run(["systemctl", "restart", "zivpn"],
                       capture_output=True, timeout=10)
    except Exception as e:
        log.warning(f"reload_pw error: {e}")

def delete_account(username: str) -> bool:
    if not Path(USERS_DB).exists():
        return False
    lines = Path(USERS_DB).read_text().splitlines()
    new_lines = [l for l in lines if not l.startswith(f"{username}|")]
    if len(new_lines) == len(lines):
        return False
    Path(USERS_DB).write_text("\n".join(new_lines) + "\n" if new_lines else "")
    # Hapus maxlogin
    if Path(MLDB).exists():
        ml = [l for l in Path(MLDB).read_text().splitlines()
              if not l.startswith(f"{username}|")]
        Path(MLDB).write_text("\n".join(ml) + "\n")
    _reload_pw()
    return True

def get_account_info(username: str) -> dict | None:
    if not Path(USERS_DB).exists():
        return None
    for line in Path(USERS_DB).read_text().splitlines():
        parts = line.split("|")
        if len(parts) >= 5 and parts[0] == username:
            ml = "2"
            if Path(MLDB).exists():
                for ml_line in Path(MLDB).read_text().splitlines():
                    if ml_line.startswith(f"{username}|"):
                        ml = ml_line.split("|")[1]
            return {
                "username": parts[0], "password": parts[1],
                "exp": parts[2], "kuota": parts[3], "note": parts[4],
                "maxlogin": ml,
                "ip": get_ip(), "domain": get_domain(), "port": get_port(),
            }
    return None

def is_admin(user_id: int) -> bool:
    return user_id in CFG.get("ADMIN_IDS", [])

# ============================================================
#  OCR — Cek Screenshot Pembayaran DANA
# ============================================================
def verify_payment_screenshot(image_path: str, expected_amount: int) -> tuple[bool, str]:
    """
    Verifikasi screenshot pembayaran DANA.
    Return (True, reason) jika valid, (False, reason) jika tidak.
    """
    if not OCR_AVAILABLE:
        # Tanpa OCR: minta admin konfirmasi manual
        return (None, "ocr_unavailable")

    try:
        img = Image.open(image_path)
        text = pytesseract.image_to_string(img, lang="ind+eng")
        text_up = text.upper()

        log.info(f"OCR result: {text[:300]}")

        # Cek kata kunci DANA
        has_dana = any(kw in text_up for kw in [
            "DANA", "BERHASIL", "SUKSES", "TRANSFER", "SELESAI",
            "SUCCESS", "PEMBAYARAN"
        ])

        # Cek nomor DANA tujuan
        dana_num = CFG.get("DANA_NUMBER", "").replace("-", "").replace(" ", "")
        has_number = dana_num in text.replace(" ", "").replace("-", "")

        # Cek nominal — cari angka di teks
        amount_strs = re.findall(r"[\d.,]+", text)
        has_amount = False
        for amt_str in amount_strs:
            try:
                amt = int(amt_str.replace(".", "").replace(",", ""))
                if amt == expected_amount:
                    has_amount = True
                    break
            except: pass

        # Cek tanggal hari ini (toleransi ±1 hari)
        today = datetime.now()
        dates_to_check = [
            today.strftime("%d/%m/%Y"), today.strftime("%d-%m-%Y"),
            today.strftime("%d %m %Y"),
            (today - timedelta(days=1)).strftime("%d/%m/%Y"),
        ]
        has_date = any(d in text for d in dates_to_check)

        if has_dana and has_number and has_amount:
            return (True, "✅ Pembayaran terverifikasi otomatis")
        elif has_dana and has_amount:
            return (True, "✅ Pembayaran terverifikasi (nominal cocok)")
        elif has_dana and has_number:
            return (False, "❌ Nominal tidak cocok dengan paket yang dipilih")
        elif not has_dana:
            return (False, "❌ Screenshot bukan dari aplikasi DANA")
        else:
            return (False, "❌ Screenshot tidak dapat diverifikasi")

    except Exception as e:
        log.error(f"OCR error: {e}")
        return (None, f"OCR error: {e}")

# ============================================================
#  FORMAT PESAN AKUN
# ============================================================
def format_akun_message(akun: dict) -> str:
    brand = CFG.get("BRAND", "OGH-ZIV")
    admin_tg = CFG.get("ADMIN_TG", "@admin")

    hari_sisa = ""
    try:
        exp_dt = datetime.strptime(akun["exp"], "%Y-%m-%d")
        sisa = (exp_dt - datetime.now()).days
        hari_sisa = f"({sisa} hari lagi)" if sisa >= 0 else "(EXPIRED)"
    except: pass

    kuota_str = "Unlimited" if str(akun.get("kuota", "0")) == "0" else akun["kuota"]

    return (
        f"🎉 <b>{brand} — Akun VPN Premium</b>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"🖥 <b>IP Publik</b>  : <code>{akun['ip']}</code>\n"
        f"🌐 <b>Host</b>      : <code>{akun['domain']}</code>\n"
        f"🔌 <b>Port</b>      : <code>{akun['port']}</code>\n"
        f"📡 <b>Obfs</b>      : <code>zivpn</code>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"👤 <b>Username</b>  : <code>{akun['username']}</code>\n"
        f"🔑 <b>Password</b>  : <code>{akun['password']}</code>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"📦 <b>Kuota</b>     : {kuota_str}\n"
        f"🔒 <b>Max Login</b> : {akun['maxlogin']} device\n"
        f"📅 <b>Expired</b>   : {akun['exp']} {hari_sisa}\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"📱 Download ZiVPN → Play Store / App Store\n"
        f"⚠️  Jangan share akun ini ke orang lain!\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"💬 Keluhan & bantuan: {admin_tg}"
    )

def format_paket_list() -> str:
    brand     = CFG.get("BRAND", "OGH-ZIV")
    dana_num  = CFG.get("DANA_NUMBER", "")
    dana_name = CFG.get("DANA_NAME", "")

    lines = [
        f"🛒 <b>{brand} — Daftar Paket UDP VPN</b>\n",
        "━━━━━━━━━━━━━━━━━━━━━━━",
        f"1️⃣  <b>3 Hari</b>   — Rp 3.000  | Unlimited | 2 device",
        f"2️⃣  <b>15 Hari</b>  — Rp 6.000  | Unlimited | 2 device",
        f"3️⃣  <b>30 Hari</b>  — Rp 10.000 | Unlimited | 2 device",
        "━━━━━━━━━━━━━━━━━━━━━━━",
        f"🎁  <b>Trial Gratis</b> — 120 Menit | 1 device",
        "━━━━━━━━━━━━━━━━━━━━━━━",
        f"💳 <b>Pembayaran via DANA</b>",
        f"📱 No. DANA : <code>{dana_num}</code>",
        f"👤 A/N      : <b>{dana_name}</b>",
    ]
    return "\n".join(lines)

# ============================================================
#  HANDLERS
# ============================================================
async def cmd_start(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    brand = CFG.get("BRAND", "OGH-ZIV")
    admin_tg = CFG.get("ADMIN_TG", "@admin")

    keyboard = [
        [InlineKeyboardButton("🛒 Beli Akun VPN",           callback_data="beli")],
        [InlineKeyboardButton("🎁 Trial Gratis 120 Menit",  callback_data="trial")],
        [InlineKeyboardButton("📋 Cek Akun Saya",           callback_data="cek_akun")],
        [InlineKeyboardButton("📞 Hubungi Admin", url=f"https://t.me/{admin_tg.lstrip('@')}")],
    ]
    if is_admin(user.id):
        keyboard.append([InlineKeyboardButton("⚙️ Admin Panel", callback_data="admin")])

    await update.message.reply_text(
        f"👋 Selamat datang di <b>{brand} VPN Bot</b>!\n\n"
        f"Bot ini membantu kamu membeli akun VPN premium dengan mudah.\n"
        f"Pembayaran via DANA — otomatis diproses setelah konfirmasi.\n\n"
        f"Pilih menu di bawah:",
        reply_markup=InlineKeyboardMarkup(keyboard),
        parse_mode="HTML"
    )

async def cb_beli(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    keyboard = [
        [InlineKeyboardButton("1️⃣  3 Hari  — Rp 3.000",  callback_data="paket_1")],
        [InlineKeyboardButton("2️⃣  15 Hari — Rp 6.000",  callback_data="paket_2")],
        [InlineKeyboardButton("3️⃣  30 Hari — Rp 10.000", callback_data="paket_3")],
        [InlineKeyboardButton("🎁  Trial Gratis 120 Menit", callback_data="trial")],
        [InlineKeyboardButton("🔙 Kembali", callback_data="back_start")],
    ]

    await query.edit_message_text(
        format_paket_list(),
        reply_markup=InlineKeyboardMarkup(keyboard),
        parse_mode="HTML"
    )

async def cb_paket(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    paket_id = query.data.split("_")[1]
    if paket_id not in PAKET:
        await query.edit_message_text("❌ Paket tidak valid.")
        return

    p = PAKET[paket_id]
    ctx.user_data["paket_id"]    = paket_id
    ctx.user_data["paket_nama"]  = p["nama"]
    ctx.user_data["paket_harga"] = p["harga"]

    brand     = CFG.get("BRAND", "OGH-ZIV")
    dana_num  = CFG.get("DANA_NUMBER", "")
    dana_name = CFG.get("DANA_NAME", "")

    await query.edit_message_text(
        f"📦 <b>Paket {p['nama']} — Rp {p['harga']:,}</b>\n\n"
        f"Silakan transfer ke:\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"💳 <b>DANA</b>\n"
        f"📱 No  : <code>{dana_num}</code>\n"
        f"👤 A/N : <b>{dana_name}</b>\n"
        f"💰 Nominal: <b>Rp {p['harga']:,}</b> (pas)\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"📸 Setelah transfer, kirim <b>screenshot bukti bayar</b> ke chat ini.\n\n"
        f"⚠️ Pastikan nominal <b>pas</b> sesuai paket!",
        parse_mode="HTML",
        reply_markup=InlineKeyboardMarkup([[
            InlineKeyboardButton("🔙 Kembali", callback_data="beli")
        ]])
    )

async def cb_trial(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    """Buat akun trial gratis 120 menit — 1x per user per hari"""
    query = update.callback_query
    await query.answer()
    user = query.from_user

    # Cek apakah user sudah pernah trial hari ini
    trial_db = Path("/etc/zivpn/trial_used.db")
    today    = datetime.now().strftime("%Y-%m-%d")
    uid_key  = f"{user.id}_{today}"

    if trial_db.exists():
        used = trial_db.read_text().splitlines()
        if uid_key in used:
            admin_tg = CFG.get("ADMIN_TG", "@admin")
            await query.edit_message_text(
                f"⛔ <b>Trial Sudah Digunakan</b>\n\n"
                f"Kamu sudah menggunakan trial gratis hari ini.\n"
                f"Trial hanya bisa digunakan <b>1x per hari</b>.\n\n"
                f"Mau lanjut? Beli paket berbayar mulai <b>Rp 3.000</b> saja!\n"
                f"Atau hubungi admin: {admin_tg}",
                parse_mode="HTML",
                reply_markup=InlineKeyboardMarkup([
                    [InlineKeyboardButton("🛒 Beli Paket", callback_data="beli")],
                    [InlineKeyboardButton("🔙 Kembali",    callback_data="back_start")],
                ])
            )
            return

    # Buat akun trial
    username = f"trial{user.id % 99999:05d}"
    # Hapus akun trial lama milik user ini jika ada
    if Path(USERS_DB).exists():
        lines = Path(USERS_DB).read_text().splitlines()
        lines = [l for l in lines if not l.startswith(f"{username}|")]
        Path(USERS_DB).write_text("\n".join(lines) + "\n" if lines else "")

    password = rand_pass(8)
    exp_dt   = datetime.now() + timedelta(minutes=TRIAL_MENIT)
    exp_str  = exp_dt.strftime("%Y-%m-%d")

    # Tulis ke users.db
    Path(USERS_DB).parent.mkdir(parents=True, exist_ok=True)
    with open(USERS_DB, "a") as f:
        f.write(f"{username}|{password}|{exp_str}|1|TRIAL-TG{user.id}\n")

    # Set maxlogin 1
    mldb = Path(MLDB)
    ml_lines = mldb.read_text().splitlines() if mldb.exists() else []
    ml_lines = [l for l in ml_lines if not l.startswith(f"{username}|")]
    ml_lines.append(f"{username}|1")
    mldb.write_text("\n".join(ml_lines) + "\n")

    _reload_pw()

    # Catat trial sudah dipakai
    with open(trial_db, "a") as f:
        f.write(uid_key + "\n")

    ip     = get_ip()
    domain = get_domain()
    port   = get_port()
    brand  = CFG.get("BRAND", "OGH-ZIV")
    admin_tg = CFG.get("ADMIN_TG", "@admin")

    # Hitung jam:menit expired
    exp_clock = exp_dt.strftime("%H:%M")
    exp_date  = exp_dt.strftime("%d/%m/%Y")

    await query.edit_message_text(
        f"🎁 <b>{brand} — Akun Trial Gratis</b>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"🖥 <b>IP Publik</b>  : <code>{ip}</code>\n"
        f"🌐 <b>Host</b>      : <code>{domain}</code>\n"
        f"🔌 <b>Port</b>      : <code>{port}</code>\n"
        f"📡 <b>Obfs</b>      : <code>zivpn</code>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"👤 <b>Username</b>  : <code>{username}</code>\n"
        f"🔑 <b>Password</b>  : <code>{password}</code>\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"⏱ <b>Durasi</b>    : 120 Menit\n"
        f"🔒 <b>Max Login</b> : 1 device\n"
        f"⏰ <b>Expired</b>   : {exp_date} pukul {exp_clock}\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"⚠️  Trial hanya bisa digunakan <b>1x per hari</b>\n"
        f"📱 Download ZiVPN → Play Store / App Store\n"
        f"💬 Keluhan: {admin_tg}",
        parse_mode="HTML",
        reply_markup=InlineKeyboardMarkup([
            [InlineKeyboardButton("🛒 Beli Paket Berbayar", callback_data="beli")],
            [InlineKeyboardButton("🔙 Menu Utama",          callback_data="back_start")],
        ])
    )

    # Notif admin
    for admin_id in CFG.get("ADMIN_IDS", []):
        try:
            await ctx.bot.send_message(
                admin_id,
                f"🎁 <b>Trial Baru</b>\n"
                f"👤 {user.full_name} (@{user.username or '-'}) | ID: {user.id}\n"
                f"🔑 {username} / {password}\n"
                f"⏰ Expired: {exp_date} {exp_clock}",
                parse_mode="HTML"
            )
        except: pass

    log.info(f"Trial dibuat: {username} untuk TG user {user.id}")
    """Handle screenshot pembayaran dari user"""
    user = update.effective_user

    # Cek apakah user sedang dalam flow pembelian
    if "paket_id" not in ctx.user_data:
        await update.message.reply_text(
            "❓ Kamu belum memilih paket.\nKetik /start untuk memulai."
        )
        return

    paket_id   = ctx.user_data["paket_id"]
    paket_info = PAKET[paket_id]

    await update.message.reply_text(
        "⏳ Memproses screenshot pembayaran kamu..."
    )

    # Download foto
    photo = update.message.photo[-1]  # resolusi terbesar
    file  = await ctx.bot.get_file(photo.file_id)
    img_path = f"/tmp/ss_payment_{user.id}_{photo.file_id[:8]}.jpg"
    await file.download_to_drive(img_path)

    # Verifikasi
    ok, reason = verify_payment_screenshot(img_path, paket_info["harga"])

    if ok is True:
        # Buat akun otomatis
        username = rand_user("ziv")
        password = rand_pass()
        akun = create_account(
            username   = username,
            password   = password,
            days       = paket_info["hari"],
            kuota      = paket_info["kuota"],
            maxlogin   = paket_info["maxlogin"],
            note       = f"TG:{user.username or user.first_name}"
        )
        ctx.user_data.clear()

        # Kirim info akun ke pembeli
        await update.message.reply_text(
            f"✅ <b>Pembayaran Terverifikasi!</b>\n\n"
            + format_akun_message(akun),
            parse_mode="HTML"
        )

        # Notif ke admin
        brand = CFG.get("BRAND", "OGH-ZIV")
        admin_notif = (
            f"💰 <b>Pesanan Baru — {brand}</b>\n"
            f"━━━━━━━━━━━━━━━━━━━\n"
            f"👤 Pembeli  : {user.full_name} (@{user.username or '-'})\n"
            f"📦 Paket    : {paket_info['nama']}\n"
            f"💰 Nominal  : Rp {paket_info['harga']:,}\n"
            f"🔑 Username : <code>{username}</code>\n"
            f"📅 Expired  : {akun['exp']}\n"
            f"✅ Status   : Otomatis dibuat"
        )
        for admin_id in CFG.get("ADMIN_IDS", []):
            try:
                await ctx.bot.send_message(admin_id, admin_notif, parse_mode="HTML")
            except: pass

        log.info(f"Akun dibuat: {username} untuk user TG {user.id}")

    elif ok is None:
        # OCR tidak tersedia → forward ke admin untuk verifikasi manual
        ctx.user_data["pending_ss_file"] = img_path
        ctx.user_data["pending_user_id"] = user.id
        ctx.user_data["pending_user_name"] = user.full_name
        ctx.user_data["pending_username"]  = user.username or "-"

        await update.message.reply_text(
            "⏳ Screenshot kamu sudah diterima.\n"
            "Admin akan memverifikasi pembayaran kamu dalam beberapa menit.\n"
            "Harap tunggu notifikasi dari bot ini ya! 🙏"
        )

        # Forward ke admin
        brand = CFG.get("BRAND", "OGH-ZIV")
        for admin_id in CFG.get("ADMIN_IDS", []):
            try:
                await ctx.bot.send_photo(
                    chat_id = admin_id,
                    photo   = open(img_path, "rb"),
                    caption = (
                        f"🧾 <b>Verifikasi Manual Diperlukan</b>\n\n"
                        f"👤 Pembeli  : {user.full_name} (@{user.username or '-'})\n"
                        f"🆔 User ID  : <code>{user.id}</code>\n"
                        f"📦 Paket    : {paket_info['nama']}\n"
                        f"💰 Nominal  : Rp {paket_info['harga']:,}\n\n"
                        f"Ketik /konfirm_{user.id} untuk approve\n"
                        f"Ketik /tolak_{user.id} untuk tolak"
                    ),
                    parse_mode="HTML"
                )
            except: pass

    else:
        # Verifikasi gagal
        admin_tg = CFG.get("ADMIN_TG", "@admin")
        await update.message.reply_text(
            f"❌ <b>Verifikasi Gagal</b>\n\n"
            f"{reason}\n\n"
            f"Pastikan:\n"
            f"• Screenshot dari aplikasi DANA\n"
            f"• Nominal transfer <b>Rp {paket_info['harga']:,}</b> (pas)\n"
            f"• Tujuan transfer nomor yang benar\n\n"
            f"Coba lagi atau hubungi admin: {admin_tg}",
            parse_mode="HTML"
        )

    # Hapus file temp
    try:
        os.remove(img_path)
    except: pass

# ============================================================
#  ADMIN: Konfirmasi Manual
# ============================================================
async def cmd_konfirm(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    if not is_admin(update.effective_user.id):
        return

    try:
        target_uid = int(update.message.text.split("_")[1])
    except:
        await update.message.reply_text("Format: /konfirm_<user_id>")
        return

    # Cari data pending di user_data (simpel: gunakan bot_data sebagai global store)
    pdata = ctx.bot_data.get(f"pending_{target_uid}")
    if not pdata:
        await update.message.reply_text("❌ Data pending tidak ditemukan.")
        return

    paket_id   = pdata.get("paket_id", "2")
    paket_info = PAKET.get(paket_id, PAKET["2"])

    username = rand_user("ziv")
    password = rand_pass()
    akun = create_account(
        username  = username,
        password  = password,
        days      = paket_info["hari"],
        kuota     = paket_info["kuota"],
        maxlogin  = paket_info["maxlogin"],
        note      = f"TG:{pdata.get('username', '-')}"
    )

    # Kirim akun ke pembeli
    try:
        await ctx.bot.send_message(
            chat_id    = target_uid,
            text       = "✅ <b>Pembayaran Dikonfirmasi Admin!</b>\n\n" + format_akun_message(akun),
            parse_mode = "HTML"
        )
    except Exception as e:
        await update.message.reply_text(f"⚠️ Tidak bisa kirim ke user: {e}")

    ctx.bot_data.pop(f"pending_{target_uid}", None)
    await update.message.reply_text(
        f"✅ Akun <code>{username}</code> berhasil dibuat dan dikirim ke user.",
        parse_mode="HTML"
    )

async def cmd_tolak(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    if not is_admin(update.effective_user.id):
        return

    try:
        target_uid = int(update.message.text.split("_")[1])
    except:
        await update.message.reply_text("Format: /tolak_<user_id>")
        return

    ctx.bot_data.pop(f"pending_{target_uid}", None)
    admin_tg = CFG.get("ADMIN_TG", "@admin")

    try:
        await ctx.bot.send_message(
            target_uid,
            f"❌ <b>Pembayaran Ditolak</b>\n\n"
            f"Screenshot pembayaran kamu tidak berhasil diverifikasi.\n"
            f"Hubungi admin untuk bantuan: {admin_tg}",
            parse_mode="HTML"
        )
    except: pass

    await update.message.reply_text("✅ Pesanan ditolak dan user telah diberitahu.")

# ============================================================
#  ADMIN PANEL
# ============================================================
async def cb_admin(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    if not is_admin(query.from_user.id):
        await query.edit_message_text("⛔ Akses ditolak!")
        return

    keyboard = [
        [InlineKeyboardButton("➕ Buat Akun Gratis", callback_data="admin_add")],
        [InlineKeyboardButton("🗑️ Hapus Akun",       callback_data="admin_del")],
        [InlineKeyboardButton("📋 List Semua Akun",   callback_data="admin_list")],
        [InlineKeyboardButton("📊 Statistik",         callback_data="admin_stat")],
        [InlineKeyboardButton("🔙 Kembali",           callback_data="back_start")],
    ]
    await query.edit_message_text(
        "⚙️ <b>Admin Panel</b>\n\nPilih aksi:",
        reply_markup=InlineKeyboardMarkup(keyboard),
        parse_mode="HTML"
    )

async def cb_admin_add(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    if not is_admin(query.from_user.id):
        return

    ctx.user_data["admin_action"] = "add_free"
    await query.edit_message_text(
        "➕ <b>Buat Akun Gratis (Admin)</b>\n\n"
        "Kirim format:\n"
        "<code>username hari kuota maxlogin</code>\n\n"
        "Contoh:\n"
        "<code>budi30 30 10 2</code>\n\n"
        "Atau ketik <code>auto</code> untuk username otomatis.",
        parse_mode="HTML"
    )

async def cb_admin_list(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    if not is_admin(query.from_user.id):
        return

    if not Path(USERS_DB).exists() or Path(USERS_DB).stat().st_size == 0:
        await query.edit_message_text(
            "📋 Belum ada akun terdaftar.",
            reply_markup=InlineKeyboardMarkup([[
                InlineKeyboardButton("🔙 Kembali", callback_data="admin")
            ]])
        )
        return

    today = datetime.now().strftime("%Y-%m-%d")
    lines_out = ["📋 <b>Daftar Akun</b>\n━━━━━━━━━━━━━━━━━━━━━━━"]
    for i, line in enumerate(Path(USERS_DB).read_text().splitlines(), 1):
        if not line.strip(): continue
        parts = line.split("|")
        if len(parts) < 3: continue
        u, p, exp = parts[0], parts[1], parts[2]
        status = "✅" if exp >= today else "❌"
        lines_out.append(f"{i}. {status} <code>{u}</code> | Exp: {exp}")
        if i >= 30:
            lines_out.append("... (terlalu banyak, max 30 ditampilkan)")
            break

    await query.edit_message_text(
        "\n".join(lines_out),
        parse_mode="HTML",
        reply_markup=InlineKeyboardMarkup([[
            InlineKeyboardButton("🔙 Kembali", callback_data="admin")
        ]])
    )

async def cb_admin_stat(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    today = datetime.now().strftime("%Y-%m-%d")
    total = aktif = expired = 0

    if Path(USERS_DB).exists():
        for line in Path(USERS_DB).read_text().splitlines():
            if not line.strip(): continue
            parts = line.split("|")
            if len(parts) >= 3:
                total += 1
                if parts[2] >= today: aktif  += 1
                else:                  expired += 1

    ip = get_ip()
    port = get_port()
    domain = get_domain()

    await query.edit_message_text(
        f"📊 <b>Statistik Server</b>\n\n"
        f"🖥 IP     : <code>{ip}</code>\n"
        f"🌐 Domain : <code>{domain}</code>\n"
        f"🔌 Port   : <code>{port}</code>\n\n"
        f"👥 Total Akun  : {total}\n"
        f"✅ Aktif       : {aktif}\n"
        f"❌ Expired     : {expired}",
        parse_mode="HTML",
        reply_markup=InlineKeyboardMarkup([[
            InlineKeyboardButton("🔙 Kembali", callback_data="admin")
        ]])
    )

async def handle_admin_text(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    """Handle teks dari admin untuk membuat akun gratis"""
    if not is_admin(update.effective_user.id):
        return

    action = ctx.user_data.get("admin_action")
    if action != "add_free":
        return

    text = update.message.text.strip()
    parts = text.split()

    if parts[0].lower() == "auto":
        username = rand_user("ziv")
        hari, kuota, maxlogin = 30, 0, 2
    elif len(parts) >= 4:
        username = parts[0]
        try:
            hari     = int(parts[1])
            kuota    = int(parts[2])
            maxlogin = int(parts[3])
        except:
            await update.message.reply_text("❌ Format salah! Contoh: budi30 30 10 2")
            return
    else:
        await update.message.reply_text("❌ Format salah! Contoh: budi30 30 10 2")
        return

    if user_exists(username):
        await update.message.reply_text(f"❌ Username <code>{username}</code> sudah ada!", parse_mode="HTML")
        return

    password = rand_pass()
    akun = create_account(
        username  = username,
        password  = password,
        days      = hari,
        kuota     = kuota,
        maxlogin  = maxlogin,
        note      = f"ADMIN-FREE"
    )

    ctx.user_data.pop("admin_action", None)

    await update.message.reply_text(
        f"✅ <b>Akun Gratis Berhasil Dibuat!</b>\n\n"
        + format_akun_message(akun),
        parse_mode="HTML"
    )

async def cb_cek_akun(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    ctx.user_data["action"] = "cek_akun"
    await query.edit_message_text(
        "🔍 <b>Cek Akun</b>\n\nKirim username kamu:",
        parse_mode="HTML",
        reply_markup=InlineKeyboardMarkup([[
            InlineKeyboardButton("🔙 Kembali", callback_data="back_start")
        ]])
    )

async def handle_text(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    action = ctx.user_data.get("action")
    admin_action = ctx.user_data.get("admin_action")

    # Admin: buat akun gratis
    if admin_action == "add_free" and is_admin(update.effective_user.id):
        await handle_admin_text(update, ctx)
        return

    # Cek akun user biasa
    if action == "cek_akun":
        username = update.message.text.strip()
        info = get_account_info(username)
        ctx.user_data.pop("action", None)

        if not info:
            await update.message.reply_text(
                f"❌ Akun <code>{username}</code> tidak ditemukan.",
                parse_mode="HTML"
            )
        else:
            await update.message.reply_text(
                format_akun_message(info),
                parse_mode="HTML"
            )
        return

    # Default
    await update.message.reply_text(
        "Ketik /start untuk memulai.",
    )

async def cb_back_start(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    user = update.effective_user
    brand = CFG.get("BRAND", "OGH-ZIV")
    admin_tg = CFG.get("ADMIN_TG", "@admin")

    keyboard = [
        [InlineKeyboardButton("🛒 Beli Akun VPN",           callback_data="beli")],
        [InlineKeyboardButton("🎁 Trial Gratis 120 Menit",  callback_data="trial")],
        [InlineKeyboardButton("📋 Cek Akun Saya",           callback_data="cek_akun")],
        [InlineKeyboardButton("📞 Hubungi Admin", url=f"https://t.me/{admin_tg.lstrip('@')}")],
    ]
    if is_admin(user.id):
        keyboard.append([InlineKeyboardButton("⚙️ Admin Panel", callback_data="admin")])

    await query.edit_message_text(
        f"👋 Selamat datang di <b>{brand} VPN Bot</b>!\n\n"
        f"Bot ini membantu kamu membeli akun VPN premium dengan mudah.\n"
        f"Pembayaran via DANA — otomatis diproses setelah konfirmasi.\n\n"
        f"Pilih menu di bawah:",
        reply_markup=InlineKeyboardMarkup(keyboard),
        parse_mode="HTML"
    )

async def cb_admin_del(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    if not is_admin(query.from_user.id):
        return

    ctx.user_data["admin_action"] = "del_akun"
    await query.edit_message_text(
        "🗑️ <b>Hapus Akun</b>\n\nKirim username yang ingin dihapus:",
        parse_mode="HTML"
    )

async def handle_admin_del(update: Update, ctx: ContextTypes.DEFAULT_TYPE):
    if ctx.user_data.get("admin_action") != "del_akun":
        return
    if not is_admin(update.effective_user.id):
        return

    username = update.message.text.strip()
    ctx.user_data.pop("admin_action", None)

    if delete_account(username):
        await update.message.reply_text(
            f"✅ Akun <code>{username}</code> berhasil dihapus.",
            parse_mode="HTML"
        )
    else:
        await update.message.reply_text(
            f"❌ Akun <code>{username}</code> tidak ditemukan.",
            parse_mode="HTML"
        )

# ============================================================
#  SETUP & RUN
# ============================================================
def write_default_config():
    """Tulis config default jika belum ada"""
    p = Path(CONFIG_FILE)
    p.parent.mkdir(parents=True, exist_ok=True)
    if not p.exists():
        p.write_text(
            "# OGH-ZIV Bot Store Config\n"
            "BOT_TOKEN=ISI_TOKEN_BOT_TELEGRAM_DI_SINI\n"
            "ADMIN_IDS=ISI_CHAT_ID_ADMIN\n"
            "DANA_NUMBER=08xxxxxxxxxx\n"
            "DANA_NAME=Nama Pemilik Dana\n"
            "BRAND=OGH-ZIV\n"
            "ADMIN_TG=@namaadmin\n"
        )
        print(f"[INFO] Config default dibuat: {CONFIG_FILE}")
        print("[INFO] Edit file tersebut lalu jalankan kembali bot ini!")

def main():
    global CFG
    write_default_config()
    CFG = load_config()

    token = CFG.get("BOT_TOKEN", "")
    if not token or token == "ISI_TOKEN_BOT_TELEGRAM_DI_SINI":
        print(f"\n[ERROR] Token bot belum diisi!")
        print(f"Edit file: {CONFIG_FILE}")
        print("Isi BOT_TOKEN dengan token dari @BotFather")
        return

    print(f"[INFO] OGH-ZIV Bot starting...")
    print(f"[INFO] Brand  : {CFG.get('BRAND')}")
    print(f"[INFO] Dana   : {CFG.get('DANA_NUMBER')}")
    print(f"[INFO] Admins : {CFG.get('ADMIN_IDS')}")
    print(f"[INFO] OCR    : {'Aktif' if OCR_AVAILABLE else 'Tidak aktif (manual mode)'}")

    app = ApplicationBuilder().token(token).build()

    # Handlers
    app.add_handler(CommandHandler("start", cmd_start))

    # Admin konfirm/tolak manual
    app.add_handler(MessageHandler(
        filters.Regex(r"^/konfirm_\d+$") & filters.User(CFG.get("ADMIN_IDS", [])),
        cmd_konfirm
    ))
    app.add_handler(MessageHandler(
        filters.Regex(r"^/tolak_\d+$") & filters.User(CFG.get("ADMIN_IDS", [])),
        cmd_tolak
    ))

    # Callback buttons
    app.add_handler(CallbackQueryHandler(cb_beli,       pattern="^beli$"))
    app.add_handler(CallbackQueryHandler(cb_trial,      pattern="^trial$"))
    app.add_handler(CallbackQueryHandler(cb_paket,      pattern="^paket_"))
    app.add_handler(CallbackQueryHandler(cb_cek_akun,   pattern="^cek_akun$"))
    app.add_handler(CallbackQueryHandler(cb_admin,      pattern="^admin$"))
    app.add_handler(CallbackQueryHandler(cb_admin_add,  pattern="^admin_add$"))
    app.add_handler(CallbackQueryHandler(cb_admin_del,  pattern="^admin_del$"))
    app.add_handler(CallbackQueryHandler(cb_admin_list, pattern="^admin_list$"))
    app.add_handler(CallbackQueryHandler(cb_admin_stat, pattern="^admin_stat$"))
    app.add_handler(CallbackQueryHandler(cb_back_start, pattern="^back_start$"))

    # Photo handler (screenshot pembayaran)
    app.add_handler(MessageHandler(filters.PHOTO, handle_photo))

    # Text handler (cek akun & admin action)
    app.add_handler(MessageHandler(
        filters.TEXT & ~filters.COMMAND, handle_text
    ))

    print("[INFO] Bot berjalan... Tekan Ctrl+C untuk berhenti.\n")
    app.run_polling(drop_pending_updates=True)

if __name__ == "__main__":
    main()
