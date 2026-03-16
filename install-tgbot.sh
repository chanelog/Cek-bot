#!/bin/bash
# ============================================================
#   OGH-ZIV Telegram Bot — Installer
#   Terintegrasi dengan OGH-ZIV Panel
#   GitHub: https://github.com/chanelog/Cek-bot
# ============================================================

R='\033[1;31m'; Y='\033[1;33m'; G='\033[1;32m'
C='\033[1;36m'; W='\033[1;37m'; N='\033[0m'
DIM='\033[2m'

BOT_FILE="/etc/zivpn/bot_store.conf"
BOT_PY="/usr/local/bin/zivpn-tgbot.py"
SVC_FILE="/etc/systemd/system/zivpn-tgbot.service"

clear
echo ""
echo -e "${C}  ╔══════════════════════════════════════════════════════╗${N}"
echo -e "${C}  ║   🤖  OGH-ZIV TELEGRAM BOT INSTALLER               ║${N}"
echo -e "${C}  ╠══════════════════════════════════════════════════════╣${N}"
echo -e "${C}  ║${N}  Auto Create Akun + Cek Screenshot Pembayaran DANA  ${C}║${N}"
echo -e "${C}  ╚══════════════════════════════════════════════════════╝${N}"
echo ""

# ── Cek root ─────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && { echo -e "${R}✘ Jalankan sebagai root!${N}"; exit 1; }

# ── Cek OS ───────────────────────────────────────────────────
[[ ! -f /etc/os-release ]] && { echo -e "${R}✘ OS tidak dikenali!${N}"; exit 1; }

# ── Install dependencies ─────────────────────────────────────
echo -e "${Y}  ➜  Menginstall dependencies...${N}"
apt-get update -qq 2>/dev/null
apt-get install -y python3 python3-pip curl wget 2>/dev/null | tail -1

# Install python-telegram-bot
echo -e "${Y}  ➜  Menginstall python-telegram-bot...${N}"
pip3 install python-telegram-bot --break-system-packages -q 2>/dev/null || \
pip3 install python-telegram-bot -q 2>/dev/null

# Install OCR (opsional, untuk cek screenshot otomatis)
echo -e "${Y}  ➜  Menginstall OCR (Tesseract)...${N}"
apt-get install -y tesseract-ocr tesseract-ocr-ind 2>/dev/null | tail -1
pip3 install pytesseract Pillow --break-system-packages -q 2>/dev/null || \
pip3 install pytesseract Pillow -q 2>/dev/null

# ── Download bot script ───────────────────────────────────────
echo -e "${Y}  ➜  Mengunduh bot script dari GitHub...${N}"
curl -Ls "https://raw.githubusercontent.com/chanelog/Socks/main/zivpn_bot.py" \
    -o "$BOT_PY" 2>/dev/null || \
wget -qO "$BOT_PY" \
    "https://raw.githubusercontent.com/chanelog/Socks/main/zivpn_bot.py" 2>/dev/null

# Jika gagal download (repo belum ada), copy dari lokal jika tersedia
if [[ ! -s "$BOT_PY" ]]; then
    echo -e "${Y}  ➜  Menggunakan bot script lokal...${N}"
    [[ -f "$(dirname "$0")/zivpn_bot.py" ]] && \
        cp "$(dirname "$0")/zivpn_bot.py" "$BOT_PY"
fi

chmod +x "$BOT_PY" 2>/dev/null

# ── Konfigurasi ───────────────────────────────────────────────
echo ""
echo -e "${C}  ════════════════════════════════════════════════════${N}"
echo -e "${C}  ⚙️   KONFIGURASI BOT${N}"
echo -e "${C}  ════════════════════════════════════════════════════${N}"
echo ""

# Load konfigurasi lama jika ada
if [[ -f "$BOT_FILE" ]]; then
    source "$BOT_FILE" 2>/dev/null
fi

echo -ne "  ${C}Bot Token${N} (dari @BotFather) [${BOT_TOKEN:--}]: "
read -r inp_token
[[ -z "$inp_token" ]] && inp_token="${BOT_TOKEN:-}"
[[ -z "$inp_token" ]] && { echo -e "${R}✘ Token tidak boleh kosong!${N}"; exit 1; }

echo -ne "  ${C}Chat ID Admin${N} (ID Telegram kamu) [${ADMIN_IDS:--}]: "
read -r inp_admin
[[ -z "$inp_admin" ]] && inp_admin="${ADMIN_IDS:-}"
[[ -z "$inp_admin" ]] && { echo -e "${R}✘ Chat ID admin tidak boleh kosong!${N}"; exit 1; }

echo -ne "  ${C}No. DANA${N} (nomor penerima pembayaran) [${DANA_NUMBER:-08xxxxxxxxxx}]: "
read -r inp_dana_num
[[ -z "$inp_dana_num" ]] && inp_dana_num="${DANA_NUMBER:-08xxxxxxxxxx}"

echo -ne "  ${C}Nama Pemilik DANA${N} [${DANA_NAME:-Nama Pemilik}]: "
read -r inp_dana_name
[[ -z "$inp_dana_name" ]] && inp_dana_name="${DANA_NAME:-Nama Pemilik}"

echo -ne "  ${C}Nama Brand${N} [${BRAND:-OGH-ZIV}]: "
read -r inp_brand
[[ -z "$inp_brand" ]] && inp_brand="${BRAND:-OGH-ZIV}"

echo -ne "  ${C}Username Telegram Admin${N} (untuk keluhan) [${ADMIN_TG:-@admin}]: "
read -r inp_admin_tg
[[ -z "$inp_admin_tg" ]] && inp_admin_tg="${ADMIN_TG:-@admin}"
[[ "$inp_admin_tg" != @* ]] && inp_admin_tg="@${inp_admin_tg}"

# ── Simpan konfigurasi ────────────────────────────────────────
mkdir -p /etc/zivpn
cat > "$BOT_FILE" << EOF
# OGH-ZIV Bot Store Config
# Dibuat: $(date "+%Y-%m-%d %H:%M:%S")
BOT_TOKEN=${inp_token}
ADMIN_IDS=${inp_admin}
DANA_NUMBER=${inp_dana_num}
DANA_NAME=${inp_dana_name}
BRAND=${inp_brand}
ADMIN_TG=${inp_admin_tg}
EOF

echo -e "${G}  ✔  Konfigurasi disimpan ke ${BOT_FILE}${N}"

# ── Buat systemd service ──────────────────────────────────────
cat > "$SVC_FILE" << EOF
[Unit]
Description=OGH-ZIV Telegram Bot
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/bin/python3 ${BOT_PY}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=PYTHONUNBUFFERED=1

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable zivpn-tgbot.service &>/dev/null
systemctl restart zivpn-tgbot.service

sleep 2

# ── Cek status ───────────────────────────────────────────────
if systemctl is-active --quiet zivpn-tgbot; then
    STATUS="${G}● RUNNING${N}"
else
    STATUS="${R}● FAILED${N}"
fi

echo ""
echo -e "${C}  ╔══════════════════════════════════════════════════════╗${N}"
echo -e "${C}  ║   ✦  INSTALASI SELESAI!                             ║${N}"
echo -e "${C}  ╠══════════════════════════════════════════════════════╣${N}"
printf  "  ${C}║${N}  %-20s : ${inp_brand}%-30s${C}║${N}\n" "Brand" ""
printf  "  ${C}║${N}  %-20s : ${inp_dana_num}%-30s${C}║${N}\n" "No. DANA" ""
printf  "  ${C}║${N}  %-20s : ${inp_dana_name}%-30s${C}║${N}\n" "A/N DANA" ""
printf  "  ${C}║${N}  %-20s : ${inp_admin_tg}%-30s${C}║${N}\n" "Admin TG" ""
echo -e "${C}  ╠══════════════════════════════════════════════════════╣${N}"
echo -e "${C}  ║${N}  Status Bot : $STATUS"
echo -e "${C}  ╠══════════════════════════════════════════════════════╣${N}"
echo -e "${C}  ║${N}  Perintah :                                          ${C}║${N}"
echo -e "${C}  ║${N}  ${DIM}systemctl status  zivpn-tgbot${N}                       ${C}║${N}"
echo -e "${C}  ║${N}  ${DIM}systemctl restart zivpn-tgbot${N}                       ${C}║${N}"
echo -e "${C}  ║${N}  ${DIM}journalctl -u zivpn-tgbot -f${N}                        ${C}║${N}"
echo -e "${C}  ╚══════════════════════════════════════════════════════╝${N}"
echo ""
echo -e "  ${G}✔  Bot Telegram OGH-ZIV berhasil diinstall!${N}"
echo -e "  ${DIM}Ketik /start di bot Telegram kamu untuk memulai.${N}"
echo ""
