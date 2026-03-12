#!/bin/bash
# ================================================================
#   OGH-ZIV BOT INSTALLER v2.0
#   Terhubung ke: https://github.com/chanelog/Socks/raw/refs/heads/main/ogh-ziv.sh
#   - Hapus file bot lama otomatis
#   - Setup DANA saat instalasi
#   - Harga default: 7hr=3rb | 15hr=6rb | 30hr=10rb
# ================================================================

DIR="/etc/zivpn"
BOTCONF="$DIR/bot.conf"
DANACONF="$DIR/dana.conf"
ADMINCONF="$DIR/admin.conf"
BOTDIR="/opt/zivpn-bot"
SVC="/etc/systemd/system/zivpn-bot.service"
BOT_PY="https://raw.githubusercontent.com/chanelog/Cek-bot/main/bot-zivpn.py"

R='\033[1;31m'; G='\033[1;32m'; Y='\033[1;33m'
A='\033[38;5;135m'; W='\033[1;37m'; D='\033[2m'; N='\033[0m'

ok()  { echo -e "  ${G}✔${N}  $*"; }
err() { echo -e "  ${R}✘${N}  $*"; }
inf() { echo -e "  ${Y}➜${N}  $*"; }

clear; echo ""
echo -e "  ${A}╔══════════════════════════════════════════════════════╗${N}"
echo -e "  ${A}║${N}  🤖  OGH-ZIV BOT INSTALLER v2.0                      ${A}║${N}"
echo -e "  ${A}║${N}  ${D}Auto Create UDP VPN — Pembayaran DANA${N}               ${A}║${N}"
echo -e "  ${A}╚══════════════════════════════════════════════════════╝${N}"
echo ""

[[ $EUID -ne 0 ]] && { err "Jalankan sebagai root!"; exit 1; }

# Cek apakah ogh-ziv sudah terinstall
if [[ ! -f /usr/local/bin/zivpn ]]; then
    echo ""
    echo -e "  ${R}╔══════════════════════════════════════════════════════╗${N}"
    echo -e "  ${R}║  ⚠️  ZiVPN belum terinstall!                          ║${N}"
    echo -e "  ${R}║  Install dulu dengan perintah:                        ║${N}"
    echo -e "  ${R}║  bash <(curl -Ls https://github.com/chanelog/Socks/  ║${N}"
    echo -e "  ${R}║           raw/refs/heads/main/ogh-ziv.sh)             ║${N}"
    echo -e "  ${R}╚══════════════════════════════════════════════════════╝${N}"
    echo ""
    echo -ne "  ${Y}Lanjut install bot tanpa zivpn? (y/N): ${N}"
    read -r lanjut
    [[ "${lanjut,,}" != "y" ]] && exit 0
fi

mkdir -p "$DIR" "$BOTDIR"

# ── Load config lama jika ada ────────────────────────────────────
BOT_TOKEN=""; CHAT_ID=""; DANA_NO=""; DANA_NAME=""; ADMIN_USERNAME=""
[[ -f "$BOTCONF"   ]] && source <(grep -E '^(BOT_TOKEN|CHAT_ID)='  "$BOTCONF")
[[ -f "$DANACONF"  ]] && source <(grep -E '^(DANA_NO|DANA_NAME)='  "$DANACONF")
[[ -f "$ADMINCONF" ]] && source <(grep -E '^ADMIN_USERNAME='        "$ADMINCONF")

# ── Input ────────────────────────────────────────────────────────
echo -e "  ${A}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo -e "  ${Y}Setup Bot Telegram${N}"
echo -e "  ${A}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo ""
echo -e "  ${D}Buka @BotFather → /newbot → salin TOKEN${N}"
echo -e "  ${D}Chat ID: buka api.telegram.org/bot<TOKEN>/getUpdates${N}"
echo ""

echo -ne "  ${Y}BOT TOKEN${N}         [${BOT_TOKEN:-kosong}]: "
read -r inp_token
[[ -z "$inp_token" ]] && inp_token="$BOT_TOKEN"
[[ -z "$inp_token" ]] && { err "Token tidak boleh kosong!"; exit 1; }

echo -ne "  ${Y}CHAT ID Admin${N}     [${CHAT_ID:-kosong}]: "
read -r inp_chatid
[[ -z "$inp_chatid" ]] && inp_chatid="$CHAT_ID"
[[ -z "$inp_chatid" ]] && { err "Chat ID tidak boleh kosong!"; exit 1; }

echo -ne "  ${Y}Username Telegram${N} [${ADMIN_USERNAME:-kosong}] (tanpa @): "
read -r inp_adminuname
[[ -z "$inp_adminuname" ]] && inp_adminuname="$ADMIN_USERNAME"
[[ -z "$inp_adminuname" ]] && { err "Username admin tidak boleh kosong!"; exit 1; }
inp_adminuname="${inp_adminuname#@}"

echo ""
echo -e "  ${A}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo -e "  ${Y}Pengaturan Pembayaran DANA${N}"
echo -e "  ${A}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo ""

echo -ne "  ${Y}Nomor DANA${N}        [${DANA_NO:--}]: "
read -r inp_dana_no
[[ -z "$inp_dana_no" ]] && inp_dana_no="$DANA_NO"
[[ -z "$inp_dana_no" ]] && { err "Nomor DANA tidak boleh kosong!"; exit 1; }

echo -ne "  ${Y}Nama A/N DANA${N}     [${DANA_NAME:--}]: "
read -r inp_dana_name
[[ -z "$inp_dana_name" ]] && inp_dana_name="$DANA_NAME"
[[ -z "$inp_dana_name" ]] && { err "Nama DANA tidak boleh kosong!"; exit 1; }

echo ""
inf "Menyimpan konfigurasi..."

# ── Simpan config ────────────────────────────────────────────────
if [[ -f "$BOTCONF" ]]; then
    grep -q "^BOT_TOKEN=" "$BOTCONF" \
        && sed -i "s|^BOT_TOKEN=.*|BOT_TOKEN=${inp_token}|"   "$BOTCONF" \
        || echo "BOT_TOKEN=${inp_token}" >> "$BOTCONF"
    grep -q "^CHAT_ID=" "$BOTCONF" \
        && sed -i "s|^CHAT_ID=.*|CHAT_ID=${inp_chatid}|"      "$BOTCONF" \
        || echo "CHAT_ID=${inp_chatid}" >> "$BOTCONF"
else
    printf "BOT_TOKEN=%s\nCHAT_ID=%s\n" "$inp_token" "$inp_chatid" > "$BOTCONF"
fi

printf "DANA_NO=%s\nDANA_NAME=%s\n"   "$inp_dana_no" "$inp_dana_name" > "$DANACONF"
printf "ADMIN_USERNAME=%s\n"           "$inp_adminuname"               > "$ADMINCONF"

ok "Konfigurasi disimpan"

# ── Paket default ────────────────────────────────────────────────
if [[ ! -s "$DIR/paket.db" ]]; then
    {
        echo "UDP 7 Hari|7|3000|0|2"
        echo "UDP 15 Hari|15|6000|0|2"
        echo "UDP 30 Hari|30|10000|0|2"
    } > "$DIR/paket.db"
    ok "Paket default: 7hr=Rp3rb | 15hr=Rp6rb | 30hr=Rp10rb"
fi

# ── Python & library ─────────────────────────────────────────────
echo ""
inf "Install Python & pyTelegramBotAPI..."
apt-get update -qq &>/dev/null
apt-get install -y -qq python3 python3-pip curl wget &>/dev/null
pip3 install pyTelegramBotAPI --quiet --break-system-packages 2>/dev/null \
    || pip3 install pyTelegramBotAPI --quiet 2>/dev/null
python3 -c "import telebot" 2>/dev/null \
    && ok "pyTelegramBotAPI siap" \
    || { err "Gagal install pyTelegramBotAPI!"; exit 1; }

# ── Hapus file bot lama ──────────────────────────────────────────
echo ""
inf "Menghapus file bot lama..."
systemctl stop zivpn-bot &>/dev/null || true
if [[ -f "$BOTDIR/bot-zivpn.py" ]]; then
    rm -f "$BOTDIR/bot-zivpn.py"
    ok "File bot lama dihapus"
else
    inf "Tidak ada file bot lama"
fi

# ── Download bot.py terbaru ──────────────────────────────────────
echo ""
inf "Download bot-zivpn.py terbaru..."
curl -Ls "$BOT_PY" -o "$BOTDIR/bot-zivpn.py" 2>/dev/null \
    || wget -qO "$BOTDIR/bot-zivpn.py" "$BOT_PY" 2>/dev/null
[[ -s "$BOTDIR/bot-zivpn.py" ]] \
    && ok "Bot berhasil didownload" \
    || { err "Gagal download bot. Cek koneksi!"; exit 1; }

# ── Systemd service ──────────────────────────────────────────────
echo ""
inf "Membuat service zivpn-bot..."
cat > "$SVC" << 'SVCEOF'
[Unit]
Description=OGH-ZIV Telegram Bot v2.0
After=network.target zivpn.service

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
    ok "Bot aktif & berjalan!"
else
    err "Bot gagal start! Log error:"
    journalctl -u zivpn-bot -n 20 --no-pager
    exit 1
fi

# ── Ringkasan ────────────────────────────────────────────────────
SERVER_IP=$(curl -s --max-time 5 https://api.ipify.org 2>/dev/null || echo "?")

echo ""
echo -e "  ${A}┌──────────────────────────────────────────────────────┐${N}"
echo -e "  ${A}│${N}  ${G}✦  BOT BERHASIL DIINSTALL!${N}                          ${A}│${N}"
echo -e "  ${A}├──────────────────┬───────────────────────────────────┤${N}"
printf  "  ${A}│${N}  IP Server        ${A}│${N}  ${W}%-33s${N}  ${A}│${N}\n" "$SERVER_IP"
printf  "  ${A}│${N}  Chat ID          ${A}│${N}  ${Y}%-33s${N}  ${A}│${N}\n" "$inp_chatid"
printf  "  ${A}│${N}  Username Admin   ${A}│${N}  ${W}@%-32s${N}  ${A}│${N}\n" "$inp_adminuname"
printf  "  ${A}│${N}  No DANA          ${A}│${N}  ${W}%-33s${N}  ${A}│${N}\n" "$inp_dana_no"
printf  "  ${A}│${N}  A/N DANA         ${A}│${N}  ${W}%-33s${N}  ${A}│${N}\n" "$inp_dana_name"
echo -e "  ${A}├──────────────────┴───────────────────────────────────┤${N}"
echo -e "  ${A}│${N}  📦 Paket: 7hr=Rp3rb | 15hr=Rp6rb | 30hr=Rp10rb    ${A}│${N}"
echo -e "  ${A}├──────────────────────────────────────────────────────┤${N}"
echo -e "  ${A}│${N}  Perintah Admin:                                     ${A}│${N}"
echo -e "  ${A}│${N}  /listorder   /confirm ID    /reject ID              ${A}│${N}"
echo -e "  ${A}│${N}  /listakun    /hapusakun USER                        ${A}│${N}"
echo -e "  ${A}│${N}  /listpaket   /newpaket      /delpaket N             ${A}│${N}"
echo -e "  ${A}│${N}  /buatakun    /setdana       /info                   ${A}│${N}"
echo -e "  ${A}│${N}  /panel  → panel VPS akun aktif + IP                 ${A}│${N}"
echo -e "  ${A}│${N}  /ip     → cek IP & port server                      ${A}│${N}"
echo -e "  ${A}├──────────────────────────────────────────────────────┤${N}"
echo -e "  ${A}│${N}  ${D}Log    : tail -f /etc/zivpn/bot.log${N}                ${A}│${N}"
echo -e "  ${A}│${N}  ${D}Restart: systemctl restart zivpn-bot${N}               ${A}│${N}"
echo -e "  ${A}│${N}  ${D}Update : bash <(curl -Ls <URL install-bot.sh>)${N}     ${A}│${N}"
echo -e "  ${A}└──────────────────────────────────────────────────────┘${N}"
echo ""
echo -e "  ${Y}💡 Perintah update bot cepat (tanpa installer):${N}"
echo -e "  ${W}systemctl stop zivpn-bot && rm -f $BOTDIR/bot-zivpn.py && curl -Ls $BOT_PY -o $BOTDIR/bot-zivpn.py && systemctl restart zivpn-bot && echo OK${N}"
echo ""
