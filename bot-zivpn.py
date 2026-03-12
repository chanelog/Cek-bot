#!/usr/bin/env python3
# ================================================================
#   OGH-ZIV BOT — Auto Create UDP VPN + Pembayaran DANA
#   Terhubung langsung ke database OGH-ZIV (/etc/zivpn/)
# ================================================================

import os, re, json, time, random, string, logging, subprocess
from datetime import datetime, timedelta
import telebot
from telebot import types

# ── Path database OGH-ZIV ───────────────────────────────────────
DIR        = "/etc/zivpn"
UDB        = f"{DIR}/users.db"
PAKETDB    = f"{DIR}/paket.db"
ORDERDB    = f"{DIR}/orders.db"
BOTCONF    = f"{DIR}/bot.conf"
DANACONF   = f"{DIR}/dana.conf"
DOMAINF    = f"{DIR}/domain.conf"
MAXLOGDB   = f"{DIR}/maxlogin.db"
CFGJSON    = f"{DIR}/config.json"
LOGFILE    = f"{DIR}/bot.log"

# ── Logging ─────────────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    handlers=[logging.FileHandler(LOGFILE), logging.StreamHandler()]
)
log = logging.getLogger(__name__)

# ================================================================
#  BACA KONFIGURASI
# ================================================================
def _read_conf(path):
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

_bc = _read_conf(BOTCONF)
_dc = _read_conf(DANACONF)

BOT_TOKEN = _bc.get("BOT_TOKEN", "")
ADMIN_ID  = int(_bc.get("CHAT_ID", "0"))

if not BOT_TOKEN:
    raise SystemExit("❌ BOT_TOKEN kosong di /etc/zivpn/bot.conf")

bot = telebot.TeleBot(BOT_TOKEN, parse_mode="Markdown")

# Variabel DANA disimpan di dict agar bisa diubah di nested func
DANA = {
    "no"  : _dc.get("DANA_NO",   "-"),
    "name": _dc.get("DANA_NAME", "-"),
}

# ── State per user ───────────────────────────────────────────────
STATE = {}   # { uid: { "step": ..., ...data... } }

# ================================================================
#  HELPER UMUM
# ================================================================
def rp(n):
    try:
        return f"{int(n):,}".replace(",", ".")
    except Exception:
        return str(n)

def order_id():
    ts   = str(int(time.time()))[-5:]
    rand = "".join(random.choices(string.digits, k=4))
    return f"ORD{ts}{rand}"

def get_domain():
    try:
        return open(DOMAINF).read().strip()
    except Exception:
        return "domain.com"

def get_port():
    try:
        cfg = json.load(open(CFGJSON))
        return cfg.get("listen", ":5667").lstrip(":")
    except Exception:
        return "5667"

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
def paket_load():
    hasil = []
    try:
        for line in open(PAKETDB):
            p = line.strip().split("|")
            if len(p) >= 5:
                hasil.append({
                    "nama"    : p[0],
                    "hari"    : int(p[1]),
                    "harga"   : int(p[2]),
                    "kuota"   : int(p[3]),
                    "maxlogin": int(p[4]),
                })
    except Exception:
        pass
    return hasil

def paket_save(lst):
    with open(PAKETDB, "w") as f:
        for p in lst:
            f.write(f"{p['nama']}|{p['hari']}|{p['harga']}|{p['kuota']}|{p['maxlogin']}\n")

def paket_init_default():
    if not os.path.exists(PAKETDB) or os.path.getsize(PAKETDB) == 0:
        with open(PAKETDB, "w") as f:
            f.write("UDP 1 Hari|1|10000|0|2\n")
            f.write("UDP 7 Hari|7|10000|0|2\n")
            f.write("UDP 30 Hari|30|10000|0|2\n")

def kuota_str(n):
    return "Unlimited" if n == 0 else f"{n} GB"

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
                    "id"      : p[0],
                    "uid"     : int(p[1]),
                    "uname"   : p[2],
                    "paket"   : p[3],
                    "hari"    : int(p[4]),
                    "harga"   : int(p[5]),
                    "kuota"   : int(p[6]),
                    "maxlogin": int(p[7]),
                    "user"    : p[8],
                    "pass"    : p[9],
                    "status"  : p[10],
                    "tgl"     : p[11],
                })
    except Exception:
        pass
    return hasil

def order_get(oid):
    for o in order_load():
        if o["id"] == oid:
            return o
    return None

def order_update_status(oid, status):
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

# ================================================================
#  DATABASE AKUN VPN (users.db OGH-ZIV)
# ================================================================
def akun_exists(username):
    try:
        for line in open(UDB):
            if line.startswith(f"{username}|"):
                return True
    except Exception:
        pass
    return False

def akun_create(o):
    """Buat akun VPN, tulis ke users.db, reload config.json, restart zivpn"""
    exp = (datetime.now() + timedelta(days=o["hari"])).strftime("%Y-%m-%d")
    note = f"Pembeli: @{o['uname']}"

    with open(UDB, "a") as f:
        f.write(f"{o['user']}|{o['pass']}|{exp}|{o['kuota']}|{note}\n")

    # maxlogin
    subprocess.run(["sed", "-i", f"/^{o['user']}|/d", MAXLOGDB], capture_output=True)
    with open(MAXLOGDB, "a") as f:
        f.write(f"{o['user']}|{o['maxlogin']}\n")

    _vpn_reload()
    return exp

def akun_hapus(username):
    subprocess.run(["sed", "-i", f"/^{username}|/d", UDB],      capture_output=True)
    subprocess.run(["sed", "-i", f"/^{username}|/d", MAXLOGDB], capture_output=True)
    _vpn_reload()

def _vpn_reload():
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
        pws   = ['"' + l.split("|")[1] + '"' for l in lines]
        cfg   = json.load(open(CFGJSON))
        cfg["auth"]["config"] = pws
        json.dump(cfg, open(CFGJSON, "w"), indent=2)
        subprocess.run(["systemctl", "restart", "zivpn"], capture_output=True)
        log.info("VPN reloaded OK")
    except Exception as e:
        log.error(f"_vpn_reload: {e}")

def dana_save():
    with open(DANACONF, "w") as f:
        f.write(f"DANA_NO={DANA['no']}\nDANA_NAME={DANA['name']}\n")

# ================================================================
#  TAMPILAN
# ================================================================
EMO = {
    "pending_payment": "⏳",
    "waiting_confirm": "🔍",
    "done"           : "🎉",
    "rejected"       : "❌",
}

def show_start(uid, fname):
    kb = types.InlineKeyboardMarkup(row_width=2)
    kb.add(
        types.InlineKeyboardButton("🛒 Beli Paket", callback_data="m_beli"),
        types.InlineKeyboardButton("📋 Cek Order",  callback_data="m_order"),
    )
    kb.add(types.InlineKeyboardButton("❓ Bantuan", callback_data="m_help"))
    send(uid,
        f"👋 Halo *{fname}*!\n\n"
        "Selamat datang di *OGH-ZIV UDP VPN Store* 🚀\n\n"
        "Pilih menu di bawah:",
        reply_markup=kb)

def show_paket(uid):
    daftar = paket_load()
    if not daftar:
        send(uid, "❌ Belum ada paket. Hubungi admin.")
        return
    text = "📦 *DAFTAR PAKET UDP VPN*\n\n"
    kb   = types.InlineKeyboardMarkup(row_width=1)
    for i, p in enumerate(daftar):
        text += (
            f"*{i+1}. {p['nama']}*\n"
            f"   ⏰ {p['hari']} hari  |  💰 Rp {rp(p['harga'])}  |  "
            f"📦 {kuota_str(p['kuota'])}  |  🔒 {p['maxlogin']} device\n\n"
        )
        kb.add(types.InlineKeyboardButton(
            f"🛒 {p['nama']} — Rp {rp(p['harga'])}",
            callback_data=f"beli_{i}",
        ))
    send(uid, text, reply_markup=kb)

def show_cek_order(uid):
    milik = [o for o in order_load() if o["uid"] == uid]
    if not milik:
        send(uid, "📋 Belum ada order.\n\nKetik /beli untuk beli paket.")
        return
    text = "📋 *ORDER KAMU*\n\n"
    for o in milik[-5:]:
        e = EMO.get(o["status"], "❓")
        text += (
            f"{e} `{o['id']}` | {o['paket']}\n"
            f"👤 `{o['user']}` | 📅 {o['tgl']}\n\n"
        )
    send(uid, text)

def show_help(uid):
    send(uid,
        "❓ *BANTUAN*\n\n"
        "*Cara Beli:*\n"
        "1. Ketik /beli → pilih paket\n"
        "2. Isi username & password VPN\n"
        "3. Transfer DANA sesuai nominal\n"
        "4. Kirim *screenshot bukti transfer*\n"
        "5. Tunggu konfirmasi admin ✅\n\n"
        "*Perintah:*\n"
        "/beli — Beli paket\n"
        "/cekorder — Cek status order\n"
        "/start — Menu utama")

# ================================================================
#  HANDLER PESAN
# ================================================================
@bot.message_handler(content_types=["photo"])
def on_photo(msg):
    uid = msg.from_user.id
    # Cari order pending milik user ini
    pending = None
    for o in order_load():
        if o["uid"] == uid and o["status"] == "pending_payment":
            pending = o
    if not pending:
        send(uid, "❓ Tidak ada order yang menunggu pembayaran.")
        return

    order_update_status(pending["id"], "waiting_confirm")
    try:
        bot.forward_message(ADMIN_ID, uid, msg.message_id)
    except Exception:
        pass
    to_admin(
        f"📸 *BUKTI PEMBAYARAN*\n\n"
        f"🆔 Order  : `{pending['id']}`\n"
        f"👤 Pembeli: @{pending['uname']}\n"
        f"📦 Paket  : {pending['paket']}\n"
        f"💰 Harga  : Rp {rp(pending['harga'])}\n"
        f"👤 Akun   : `{pending['user']}`\n"
        f"🔑 Pass   : `{pending['pass']}`\n\n"
        f"✅ `/confirm {pending['id']}`\n"
        f"❌ `/reject {pending['id']}`"
    )
    send(uid, "✅ Bukti diterima! Admin sedang memverifikasi. Tunggu sebentar 😊")


@bot.message_handler(func=lambda m: True)
def on_message(msg):
    uid  = msg.from_user.id
    text = (msg.text or "").strip()

    if uid in STATE:
        _handle_state(msg, text)
        return

    cmd = text.split()[0] if text else ""

    if cmd in ("/start", "/menu"):
        show_start(uid, msg.from_user.first_name)

    elif cmd == "/beli":
        show_paket(uid)

    elif cmd == "/cekorder":
        show_cek_order(uid)

    elif cmd == "/help":
        show_help(uid)

    elif is_admin(uid) and text.startswith("/confirm "):
        _cmd_confirm(uid, text[9:].strip())

    elif is_admin(uid) and text.startswith("/reject "):
        _cmd_reject(uid, text[8:].strip())

    elif is_admin(uid) and cmd == "/listorder":
        _cmd_listorder(uid)

    elif is_admin(uid) and cmd == "/listakun":
        _cmd_listakun(uid)

    elif is_admin(uid) and text.startswith("/hapusakun "):
        u = text[11:].strip()
        akun_hapus(u)
        send(uid, f"✅ Akun `{u}` berhasil dihapus.")

    elif is_admin(uid) and cmd == "/listpaket":
        _cmd_listpaket(uid)

    elif is_admin(uid) and cmd == "/newpaket":
        STATE[uid] = {"step": "np_nama"}
        send(uid, "📦 Masukkan *nama paket* baru:\nContoh: `UDP 30 Hari`")

    elif is_admin(uid) and text.startswith("/delpaket "):
        try:
            idx = int(text[10:].strip()) - 1
            _cmd_delpaket(uid, idx)
        except Exception:
            send(uid, "❌ Format: `/delpaket 1`")

    elif is_admin(uid) and cmd == "/setdana":
        STATE[uid] = {"step": "set_dana_no"}
        send(uid, f"💳 No DANA saat ini: `{DANA['no']}`\n\nMasukkan *nomor DANA baru*:")

    elif is_admin(uid) and cmd == "/broadcast":
        STATE[uid] = {"step": "broadcast"}
        send(uid, "📢 Ketik pesan broadcast:")

    elif is_admin(uid) and cmd == "/info":
        _cmd_info(uid)

    else:
        show_start(uid, msg.from_user.first_name)


# ================================================================
#  STATE MACHINE
# ================================================================
def _handle_state(msg, text):
    uid   = msg.from_user.id
    state = STATE[uid]
    step  = state["step"]

    # ─── Alur beli ─────────────────────────────────────────────
    if step == "input_username":
        if not re.match(r"^[a-zA-Z0-9_]{3,20}$", text):
            send(uid, "❌ Username tidak valid!\nGunakan 3-20 karakter (huruf/angka/underscore).")
            return
        if akun_exists(text):
            send(uid, "❌ Username sudah dipakai, coba yang lain.")
            return
        state["vuser"] = text
        state["step"]  = "input_password"
        STATE[uid] = state
        send(uid, f"✅ Username: `{text}`\n\n🔑 Masukkan *password* (min 6 karakter):")

    elif step == "input_password":
        if len(text) < 6:
            send(uid, "❌ Password minimal 6 karakter!")
            return

        state["vpass"] = text
        STATE.pop(uid, None)

        daftar = paket_load()
        p   = daftar[state["pidx"]]
        oid = order_id()
        ks  = kuota_str(p["kuota"])
        uname = msg.from_user.username or str(uid)

        o = {
            "id"      : oid,
            "uid"     : uid,
            "uname"   : uname,
            "paket"   : p["nama"],
            "hari"    : p["hari"],
            "harga"   : p["harga"],
            "kuota"   : p["kuota"],
            "maxlogin": p["maxlogin"],
            "user"    : state["vuser"],
            "pass"    : state["vpass"],
            "status"  : "pending_payment",
            "tgl"     : datetime.now().strftime("%Y-%m-%d %H:%M"),
        }
        order_save(o)

        send(uid,
            f"🛒 *ORDER DIBUAT!*\n\n"
            f"🆔 Order ID : `{oid}`\n"
            f"📦 Paket    : {p['nama']}\n"
            f"👤 Username : `{state['vuser']}`\n"
            f"🔑 Password : `{state['vpass']}`\n"
            f"📦 Kuota    : {ks}\n"
            f"📅 Masa     : {p['hari']} hari\n\n"
            f"💳 *BAYAR VIA DANA*\n"
            f"┌──────────────────────\n"
            f"│ No    : `{DANA['no']}`\n"
            f"│ A/N   : *{DANA['name']}*\n"
            f"│ Nominal: *Rp {rp(p['harga'])}*\n"
            f"└──────────────────────\n\n"
            f"📸 Kirim *screenshot bukti transfer* setelah bayar.\n"
            f"⏰ Order batal otomatis jika tidak dibayar dalam 30 menit."
        )
        to_admin(
            f"🔔 *ORDER BARU*\n\n"
            f"🆔 Order  : `{oid}`\n"
            f"👤 Pembeli: @{uname} (`{uid}`)\n"
            f"📦 Paket  : {p['nama']}\n"
            f"💰 Harga  : Rp {rp(p['harga'])}\n"
            f"👤 Akun   : `{state['vuser']}`\n"
            f"🔑 Pass   : `{state['vpass']}`\n\n"
            f"Menunggu bukti pembayaran..."
        )

    # ─── Set DANA ───────────────────────────────────────────────
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
            f"✅ *DANA disimpan!*\n\n"
            f"💳 No  : `{DANA['no']}`\n"
            f"👤 A/N : *{DANA['name']}*"
        )

    # ─── Tambah paket ───────────────────────────────────────────
    elif step == "np_nama":
        state["np_nama"] = text
        state["step"]    = "np_hari"
        STATE[uid] = state
        send(uid, f"📦 Nama: *{text}*\n\n⏰ Masukkan *masa aktif (hari)*:")

    elif step == "np_hari":
        if not text.isdigit() or int(text) < 1:
            send(uid, "❌ Masukkan angka hari yang valid!")
            return
        state["np_hari"] = int(text)
        state["step"]    = "np_harga"
        STATE[uid] = state
        send(uid, f"⏰ Masa: *{text} hari*\n\n💰 Masukkan *harga (Rupiah)*:\nContoh: `10000`")

    elif step == "np_harga":
        if not text.isdigit():
            send(uid, "❌ Masukkan angka harga yang valid!")
            return
        state["np_harga"] = int(text)
        state["step"]     = "np_kuota"
        STATE[uid] = state
        send(uid, f"💰 Harga: *Rp {rp(text)}*\n\n📦 Kuota GB (0 = unlimited):")

    elif step == "np_kuota":
        if not text.isdigit():
            send(uid, "❌ Masukkan angka kuota yang valid!")
            return
        state["np_kuota"] = int(text)
        state["step"]     = "np_maxlogin"
        STATE[uid] = state
        send(uid, f"📦 Kuota: *{kuota_str(int(text))}*\n\n🔒 Max login device:")

    elif step == "np_maxlogin":
        if not text.isdigit() or int(text) < 1:
            send(uid, "❌ Masukkan angka max login yang valid!")
            return
        daftar = paket_load()
        daftar.append({
            "nama"    : state["np_nama"],
            "hari"    : state["np_hari"],
            "harga"   : state["np_harga"],
            "kuota"   : state["np_kuota"],
            "maxlogin": int(text),
        })
        paket_save(daftar)
        STATE.pop(uid, None)
        send(uid,
            f"✅ *Paket ditambahkan!*\n\n"
            f"📦 {state['np_nama']}\n"
            f"⏰ {state['np_hari']} hari | 💰 Rp {rp(state['np_harga'])} | "
            f"📦 {kuota_str(state['np_kuota'])} | 🔒 {text} device"
        )

    # ─── Broadcast ──────────────────────────────────────────────
    elif step == "broadcast":
        STATE.pop(uid, None)
        to_admin(f"📢 *BROADCAST*\n\n{text}")
        send(uid, "✅ Broadcast dikirim!")


# ================================================================
#  CALLBACK (tombol inline)
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
    elif data.startswith("beli_"):
        idx    = int(data[5:])
        daftar = paket_load()
        if idx < 0 or idx >= len(daftar):
            return
        p = daftar[idx]
        STATE[uid] = {"step": "input_username", "pidx": idx}
        send(uid,
            f"✅ *{p['nama']}* dipilih\n\n"
            f"⏰ {p['hari']} hari  |  💰 Rp {rp(p['harga'])}  |  "
            f"📦 {kuota_str(p['kuota'])}  |  🔒 {p['maxlogin']} device\n\n"
            f"👤 Masukkan *username* untuk akun VPN:\n"
            f"_(3-20 karakter, huruf/angka/underscore)_"
        )


# ================================================================
#  ADMIN COMMANDS
# ================================================================
def _cmd_confirm(uid, oid):
    o = order_get(oid)
    if not o:
        send(uid, f"❌ Order tidak ditemukan: `{oid}`")
        return
    if o["status"] == "done":
        send(uid, "⚠️ Order sudah selesai.")
        return

    exp = akun_create(o)
    order_update_status(oid, "done")

    send(o["uid"],
        f"🎉 *PEMBAYARAN DIKONFIRMASI!*\n\n"
        f"Akun VPN UDP kamu sudah aktif:\n\n"
        f"👤 Username : `{o['user']}`\n"
        f"🔑 Password : `{o['pass']}`\n"
        f"🌐 Host     : `{get_domain()}`\n"
        f"🔌 Port     : `{get_port()}`\n"
        f"📡 Protocol : `zivpn`\n"
        f"📦 Kuota    : {kuota_str(o['kuota'])}\n"
        f"🔒 MaxLogin : {o['maxlogin']} device\n"
        f"📅 Expired  : {exp}\n\n"
        f"📱 Download *ZiVPN* di Play Store / App Store\n"
        f"⚠️ Jangan share akun ke orang lain!"
    )
    send(uid, f"✅ Order `{oid}` dikonfirmasi! Akun *{o['user']}* aktif.")
    log.info(f"confirmed order={oid} akun={o['user']}")


def _cmd_reject(uid, oid):
    o = order_get(oid)
    if not o:
        send(uid, f"❌ Order tidak ditemukan: `{oid}`")
        return
    order_update_status(oid, "rejected")
    send(o["uid"], f"❌ Order `{oid}` ditolak.\nHubungi admin untuk info lebih lanjut.")
    send(uid, f"✅ Order `{oid}` ditolak.")


def _cmd_listorder(uid):
    rows = order_load()
    if not rows:
        send(uid, "📋 Belum ada order.")
        return
    text = "📋 *LIST ORDER*\n\n"
    for o in rows[-10:]:
        e = EMO.get(o["status"], "❓")
        text += f"{e} `{o['id']}` | @{o['uname']} | {o['paket']} | Rp {rp(o['harga'])}\n"
    text += "\n✅ `/confirm ID`   ❌ `/reject ID`"
    send(uid, text)


def _cmd_listakun(uid):
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
    except Exception:
        lines = []
    if not lines:
        send(uid, "👤 Belum ada akun.")
        return
    today = datetime.now().strftime("%Y-%m-%d")
    text  = "👤 *LIST AKUN VPN*\n\n"
    for line in lines[:25]:
        p  = line.split("|")
        if len(p) < 3:
            continue
        st = "✅" if p[2] >= today else "❌"
        text += f"{st} `{p[0]}` | exp: {p[2]}\n"
    if len(lines) > 25:
        text += f"\n_...dan {len(lines)-25} lainnya_"
    send(uid, text)


def _cmd_listpaket(uid):
    daftar = paket_load()
    if not daftar:
        send(uid, "📦 Belum ada paket.")
        return
    text = "📦 *LIST PAKET*\n\n"
    for i, p in enumerate(daftar):
        text += (
            f"*{i+1}. {p['nama']}*\n"
            f"   ⏰ {p['hari']} hari | 💰 Rp {rp(p['harga'])} | "
            f"📦 {kuota_str(p['kuota'])} | 🔒 {p['maxlogin']} device\n"
            f"   Hapus: `/delpaket {i+1}`\n\n"
        )
    send(uid, text)


def _cmd_delpaket(uid, idx):
    daftar = paket_load()
    if idx < 0 or idx >= len(daftar):
        send(uid, "❌ Nomor paket tidak valid.")
        return
    nama = daftar[idx]["nama"]
    daftar.pop(idx)
    paket_save(daftar)
    send(uid, f"✅ Paket *{nama}* dihapus.")


def _cmd_info(uid):
    try:
        lines = [l.strip() for l in open(UDB) if l.strip()]
    except Exception:
        lines = []
    today   = datetime.now().strftime("%Y-%m-%d")
    expired = sum(1 for l in lines if len(l.split("|")) >= 3 and l.split("|")[2] < today)
    rows    = order_load()
    pending = sum(1 for o in rows if o["status"] in ("pending_payment", "waiting_confirm"))
    send(uid,
        f"📊 *INFO PANEL*\n\n"
        f"👤 Total Akun  : {len(lines)}\n"
        f"❌ Expired     : {expired}\n"
        f"📋 Total Order : {len(rows)}\n"
        f"⏳ Pending     : {pending}\n"
        f"💳 DANA        : `{DANA['no']}` ({DANA['name']})\n"
        f"🌐 Domain      : `{get_domain()}`\n"
        f"🔌 Port        : `{get_port()}`"
    )


# ================================================================
#  MAIN
# ================================================================
if __name__ == "__main__":
    os.makedirs(DIR, exist_ok=True)
    paket_init_default()
    log.info(f"Bot start — admin={ADMIN_ID} dana={DANA['no']}")
    bot.infinity_polling(timeout=30, long_polling_timeout=30)
