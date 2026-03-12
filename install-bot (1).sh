#!/bin/bash
# ================================================================
#   OGH-ZIV BOT INSTALLER
#   Install bot Telegram Python — Auto Create UDP VPN + DANA
# ================================================================

DIR="/etc/zivpn"
BOTCONF="$DIR/bot.conf"
DANACONF="$DIR/dana.conf"
BOTDIR="/opt/zivpn-bot"
SVC="/etc/systemd/system/zivpn-bot.service"
BOT_PY="https://raw.githubusercontent.com/chanelog/Cek-bot/main/bot-zivpn.py"

R='\033[1;31m'; G='\033[1;32m'; Y='\033[1;33m'
A='\033[38;5;135m'; W='\033[1;37m'; D='\033[2m'; N='\033[0m'

ok()  { echo -e "  ${G}✔${N}  $*"; }
err() { echo -e "  ${R}✘${N}  $*"; }
inf() { echo -e "  ${Y}➜${N}  $*"; }
ttl() { echo -e "  ${A}$*${N}"; }

clear; echo ""
ttl "╔══════════════════════════════════════════════════════╗"
ttl "║  🤖  OGH-ZIV BOT INSTALLER                           ║"
ttl "║  ${D}Bot Telegram — Auto Create UDP VPN${A}                 ║"
ttl "╚══════════════════════════════════════════════════════╝"
echo ""

[[ $EUID -ne 0 ]] && { err "Jalankan sebagai root!"; exit 1; }
mkdir -p "$DIR" "$BOTDIR"

# ── Load config lama ─────────────────────────────────────────────
BOT_TOKEN=""; CHAT_ID=""; DANA_NO=""; DANA_NAME=""
[[ -f "$BOTCONF"  ]] && source <(grep -E '^(BOT_TOKEN|CHAT_ID)=' "$BOTCONF")
[[ -f "$DANACONF" ]] && source <(grep -E '^(DANA_NO|DANA_NAME)=' "$DANACONF")

# ── Input ────────────────────────────────────────────────────────
ttl "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${Y}Konfigurasi Bot Telegram${N}"
ttl "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "  ${D}Buka @BotFather → /newbot → salin TOKEN${N}"
echo -e "  ${D}Chat ID: buka api.telegram.org/bot<TOKEN>/getUpdates${N}"
echo ""

echo -ne "  ${Y}BOT TOKEN${N}      [${BOT_TOKEN:-kosong}]: "
read -r inp_token
[[ -z "$inp_token" ]] && inp_token="$BOT_TOKEN"
[[ -z "$inp_token" ]] && { err "Token tidak boleh kosong!"; exit 1; }

echo -ne "  ${Y}CHAT ID Admin${N}  [${CHAT_ID:-kosong}]: "
read -r inp_chatid
[[ -z "$inp_chatid" ]] && inp_chatid="$CHAT_ID"
[[ -z "$inp_chatid" ]] && { err "Chat ID tidak boleh kosong!"; exit 1; }

echo ""
ttl "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${Y}Pengaturan Pembayaran DANA${N}"
ttl "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -ne "  ${Y}No DANA${N}        [${DANA_NO:--}]: "
read -r inp_dana_no
[[ -z "$inp_dana_no" ]] && inp_dana_no="$DANA_NO"

echo -ne "  ${Y}Nama A/N DANA${N}  [${DANA_NAME:--}]: "
read -r inp_dana_name
[[ -z "$inp_dana_name" ]] && inp_dana_name="$DANA_NAME"

echo ""
inf "Menyimpan konfigurasi..."

# Simpan / update bot.conf (pertahankan baris lain)
if [[ -f "$BOTCONF" ]]; then
    grep -q "^BOT_TOKEN=" "$BOTCONF" \
        && sed -i "s|^BOT_TOKEN=.*|BOT_TOKEN=${inp_token}|"    "$BOTCONF" \
        || echo "BOT_TOKEN=${inp_token}" >> "$BOTCONF"
    grep -q "^CHAT_ID=" "$BOTCONF" \
        && sed -i "s|^CHAT_ID=.*|CHAT_ID=${inp_chatid}|"       "$BOTCONF" \
        || echo "CHAT_ID=${inp_chatid}"  >> "$BOTCONF"
else
    printf "BOT_TOKEN=%s\nCHAT_ID=%s\n" "$inp_token" "$inp_chatid" > "$BOTCONF"
fi

# Simpan dana.conf
printf "DANA_NO=%s\nDANA_NAME=%s\n" "$inp_dana_no" "$inp_dana_name" > "$DANACONF"
ok "Konfigurasi disimpan"

# ── Python & library ─────────────────────────────────────────────
echo ""
inf "Install Python & pyTelegramBotAPI..."
apt-get update -qq &>/dev/null
apt-get install -y -qq python3 python3-pip &>/dev/null
pip3 install pyTelegramBotAPI --quiet --break-system-packages 2>/dev/null \
    || pip3 install pyTelegramBotAPI --quiet 2>/dev/null
python3 -c "import telebot" 2>/dev/null \
    && ok "pyTelegramBotAPI siap" \
    || { err "Gagal install pyTelegramBotAPI!"; exit 1; }

# ── Download bot.py ──────────────────────────────────────────────
echo ""
inf "Download bot-zivpn.py..."
curl -Ls "$BOT_PY" -o "$BOTDIR/bot-zivpn.py" 2>/dev/null \
    || wget -qO "$BOTDIR/bot-zivpn.py" "$BOT_PY" 2>/dev/null
[[ -s "$BOTDIR/bot-zivpn.py" ]] \
    && ok "Bot berhasil didownload" \
    || { err "Gagal download bot. Cek koneksi!"; exit 1; }

# ── Default paket jika belum ada ─────────────────────────────────
if [[ ! -s "$DIR/paket.db" ]]; then
    {
        echo "UDP 1 Hari|1|10000|0|2"
        echo "UDP 7 Hari|7|10000|0|2"
        echo "UDP 30 Hari|30|10000|0|2"
    } > "$DIR/paket.db"
    ok "Paket default dibuat (Rp 10.000/paket)"
fi

# ── Systemd service ──────────────────────────────────────────────
echo ""
inf "Membuat service zivpn-bot..."
cat > "$SVC" << 'SVCEOF'
[Unit]
Description=OGH-ZIV Telegram Bot
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/zivpn-bot
ExecStart=/usr/bin/python3 /opt/zivpn-bot/bot-zivpn.py
Restart=always
RestartSec=5
StandardOutput=append:/etc/zivpn/bot.log
StandardError=append:/etc/zivpn/bot.log

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable zivpn-bot &>/dev/null
systemctl restart zivpn-bot
sleep 2

if systemctl is-active --quiet zivpn-bot; then
    ok "Bot berjalan!"
else
    err "Bot gagal start!"
    echo ""
    echo -e "  Cek error:"
    journalctl -u zivpn-bot -n 15 --no-pager
    exit 1
fi

# ── Summary ──────────────────────────────────────────────────────
echo ""
ttl "┌──────────────────────────────────────────────────────┐"
ttl "│  ✦  BOT BERHASIL DIINSTALL!                          │"
ttl "├────────────────┬─────────────────────────────────────┤"
printf "  ${A}│${N}  Chat ID       ${A}│${N}  ${Y}%-35s${N}  ${A}│${N}\n" "$inp_chatid"
printf "  ${A}│${N}  No DANA       ${A}│${N}  ${W}%-35s${N}  ${A}│${N}\n" "${inp_dana_no:--}"
printf "  ${A}│${N}  A/N DANA      ${A}│${N}  ${W}%-35s${N}  ${A}│${N}\n" "${inp_dana_name:--}"
ttl "├────────────────┴─────────────────────────────────────┤"
ttl "│  Perintah Admin:                                     │"
ttl "│  /listorder  /confirm ID  /reject ID                 │"
ttl "│  /listakun   /hapusakun USERNAME                     │"
ttl "│  /listpaket  /newpaket    /delpaket N                │"
ttl "│  /setdana    /broadcast   /info                      │"
ttl "├──────────────────────────────────────────────────────┤"
ttl "│  Log    : tail -f /etc/zivpn/bot.log                 │"
ttl "│  Restart: systemctl restart zivpn-bot                │"
ttl "│  Status : systemctl status  zivpn-bot                │"
ttl "└──────────────────────────────────────────────────────┘"
echo ""
