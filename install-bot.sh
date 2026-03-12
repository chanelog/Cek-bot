#!/bin/bash
# ============================================================
#   OGH-ZIV Telegram Bot — Auto Installer
#   Jalankan: bash install-bot.sh
#   Support : Debian & Ubuntu (semua versi)
# ============================================================

RED='\033[1;31m'; GREEN='\033[1;32m'; YELLOW='\033[1;33m'
CYAN='\033[1;36m'; PURPLE='\033[1;35m'; NC='\033[0m'; BOLD='\033[1m'; DIM='\033[2m'

ok()    { echo -e "  ${GREEN}✔${NC}  $*"; }
inf()   { echo -e "  ${CYAN}➜${NC}  $*"; }
warn()  { echo -e "  ${YELLOW}⚠${NC}  $*"; }
err()   { echo -e "  ${RED}✘${NC}  $*"; }
step()  { echo -e "\n  ${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; \
          echo -e "  ${CYAN}[${NC} $* ${CYAN}]${NC}"; \
          echo -e "  ${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"; }

GO_VERSION="1.22.4"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"
GO_INSTALL_DIR="/usr/local"
GO_BIN="${GO_INSTALL_DIR}/go/bin/go"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

clear
echo ""
echo -e "  ${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "  ${CYAN}║${NC}   ${BOLD}🚀  OGH-ZIV — FULL AUTO INSTALLER${NC}                   ${CYAN}║${NC}"
echo -e "  ${CYAN}╠══════════════════════════════════════════════════════════╣${NC}"
echo -e "  ${CYAN}║${NC}  ${DIM}① Dependensi  ② Golang  ③ ZiVPN  ④ Bot  ⑤ Admin${NC}     ${CYAN}║${NC}"
echo -e "  ${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# ── CEK ROOT ────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && { err "Jalankan sebagai root! (sudo bash install-bot.sh)"; exit 1; }

# ── CEK OS ──────────────────────────────────────────────────
if [[ ! -f /etc/os-release ]]; then
    err "OS tidak dikenali!"; exit 1
fi
source /etc/os-release 2>/dev/null
OS_ID=$(echo "${ID}" | tr '[:upper:]' '[:lower:]')
if [[ "$OS_ID" != "debian" && "$OS_ID" != "ubuntu" ]] && \
   [[ "${ID_LIKE:-}" != *"debian"* && "${ID_LIKE:-}" != *"ubuntu"* ]]; then
    err "OS tidak didukung: ${PRETTY_NAME:-$ID}"
    err "Script ini hanya untuk Debian & Ubuntu."
    exit 1
fi
ok "OS: ${PRETTY_NAME:-$ID}"

# ── CEK ARSITEKTUR ──────────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  GO_ARCH="amd64" ;;
    aarch64) GO_ARCH="arm64"; GO_TARBALL="go${GO_VERSION}.linux-arm64.tar.gz"; GO_URL="https://go.dev/dl/${GO_TARBALL}" ;;
    armv7l)  GO_ARCH="armv6l"; GO_TARBALL="go${GO_VERSION}.linux-armv6l.tar.gz"; GO_URL="https://go.dev/dl/${GO_TARBALL}" ;;
    *)       err "Arsitektur tidak didukung: $ARCH"; exit 1 ;;
esac
ok "Arsitektur: $ARCH ($GO_ARCH)"

# ── CEK PANEL ───────────────────────────────────────────────
if [[ ! -f /etc/zivpn/bot.conf ]]; then
    err "Panel OGH-ZIV belum dikonfigurasi!"
    warn "Jalankan panel dulu dan setup Telegram Bot via menu (Telegram Bot → Setup)."
    exit 1
fi
BOT_TOKEN=$(grep "^BOT_TOKEN=" /etc/zivpn/bot.conf 2>/dev/null | cut -d'=' -f2 | tr -d ' ')
if [[ -z "$BOT_TOKEN" ]]; then
    err "BOT_TOKEN kosong di /etc/zivpn/bot.conf"
    warn "Setup bot Telegram dulu via panel."
    exit 1
fi
ok "Bot token ditemukan: ...${BOT_TOKEN: -8}"

# ── CEK FILE BOT SOURCE ─────────────────────────────────────
BOT_SRC="${SCRIPT_DIR}/bot-zivpn.go"
if [[ ! -f "$BOT_SRC" ]]; then
    err "File bot-zivpn.go tidak ditemukan di: ${SCRIPT_DIR}/"
    err "Pastikan bot-zivpn.go ada di folder yang sama dengan install-bot.sh"
    exit 1
fi
ok "Source bot: ${BOT_SRC}"

# ════════════════════════════════════════════════════════════
# STEP 1 — INSTALL DEPENDENSI DASAR
# ════════════════════════════════════════════════════════════
step "STEP 1 — Install Dependensi"

inf "Update package list..."
apt-get update -qq 2>/dev/null
ok "Package list diupdate"

inf "Install curl, wget, tar, git..."
apt-get install -y -qq curl wget tar git ca-certificates 2>/dev/null
ok "Dependensi dasar terpasang"

# ════════════════════════════════════════════════════════════
# STEP 2 — INSTALL / UPDATE GOLANG
# ════════════════════════════════════════════════════════════
step "STEP 2 — Install Golang ${GO_VERSION}"

install_go_official() {
    inf "Download Go ${GO_VERSION} dari go.dev..."
    inf "URL: ${GO_URL}"
    echo ""

    # Hapus file lama jika ada
    rm -f /tmp/${GO_TARBALL}

    # Download dengan progress
    wget --progress=bar:force:noscroll \
         --tries=3 \
         --timeout=120 \
         -O /tmp/${GO_TARBALL} \
         "${GO_URL}" 2>&1

    if [[ ! -s /tmp/${GO_TARBALL} ]]; then
        warn "wget gagal, coba curl..."
        curl -L --progress-bar \
             --retry 3 \
             --max-time 120 \
             -o /tmp/${GO_TARBALL} \
             "${GO_URL}"
    fi

    if [[ ! -s /tmp/${GO_TARBALL} ]]; then
        err "Download gagal!"
        return 1
    fi

    local SIZE
    SIZE=$(du -sh /tmp/${GO_TARBALL} 2>/dev/null | cut -f1)
    ok "Download selesai (${SIZE})"

    # Verifikasi tarball
    inf "Verifikasi file..."
    if ! tar -tzf /tmp/${GO_TARBALL} &>/dev/null; then
        err "File rusak atau bukan tarball valid!"
        rm -f /tmp/${GO_TARBALL}
        return 1
    fi
    ok "File valid"

    # Backup instalasi Go lama jika ada
    if [[ -d "${GO_INSTALL_DIR}/go" ]]; then
        inf "Backup Go lama → ${GO_INSTALL_DIR}/go.bak"
        rm -rf "${GO_INSTALL_DIR}/go.bak"
        mv "${GO_INSTALL_DIR}/go" "${GO_INSTALL_DIR}/go.bak"
    fi

    # Ekstrak
    inf "Ekstrak ke ${GO_INSTALL_DIR}..."
    tar -C "${GO_INSTALL_DIR}" -xzf /tmp/${GO_TARBALL}
    if [[ $? -ne 0 ]]; then
        err "Ekstrak gagal!"
        # Restore backup jika ada
        [[ -d "${GO_INSTALL_DIR}/go.bak" ]] && mv "${GO_INSTALL_DIR}/go.bak" "${GO_INSTALL_DIR}/go"
        return 1
    fi

    # Hapus backup lama
    rm -rf "${GO_INSTALL_DIR}/go.bak"
    # Hapus tarball
    rm -f /tmp/${GO_TARBALL}

    ok "Go berhasil diekstrak ke ${GO_INSTALL_DIR}/go"
    return 0
}

setup_go_path() {
    local GO_BIN_DIR="${GO_INSTALL_DIR}/go/bin"

    # Export untuk sesi ini
    export PATH="${GO_BIN_DIR}:${PATH}"
    export GOROOT="${GO_INSTALL_DIR}/go"
    export GOPATH="${HOME}/go"

    # Tambah ke /etc/profile.d/ (global, semua user)
    cat > /etc/profile.d/golang.sh << GOPATH_EOF
#!/bin/bash
export GOROOT=${GO_INSTALL_DIR}/go
export GOPATH=\${HOME}/go
export PATH=\${PATH}:${GO_BIN_DIR}:\${GOPATH}/bin
GOPATH_EOF
    chmod +x /etc/profile.d/golang.sh

    # Tambah ke /root/.bashrc
    if ! grep -q "GOROOT" /root/.bashrc 2>/dev/null; then
        cat >> /root/.bashrc << BASHRC_EOF

# Golang
export GOROOT=${GO_INSTALL_DIR}/go
export GOPATH=\${HOME}/go
export PATH=\${PATH}:${GO_BIN_DIR}:\${GOPATH}/bin
BASHRC_EOF
    fi

    # Tambah ke /root/.profile
    if ! grep -q "GOROOT" /root/.profile 2>/dev/null; then
        cat >> /root/.profile << PROFILE_EOF

# Golang
export GOROOT=${GO_INSTALL_DIR}/go
export GOPATH=\${HOME}/go
export PATH=\${PATH}:${GO_BIN_DIR}:\${GOPATH}/bin
PROFILE_EOF
    fi

    # Symlink ke /usr/local/bin agar bisa dipanggil langsung
    ln -sf "${GO_BIN_DIR}/go" /usr/local/bin/go 2>/dev/null
    ln -sf "${GO_BIN_DIR}/gofmt" /usr/local/bin/gofmt 2>/dev/null

    ok "PATH Go dikonfigurasi"
}

# Cek apakah Go sudah ada dan versi cukup baru
GO_CURRENT=""
if command -v go &>/dev/null; then
    GO_CURRENT=$(go version 2>/dev/null | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
fi
if [[ -x "${GO_BIN}" ]]; then
    GO_CURRENT=$(${GO_BIN} version 2>/dev/null | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
fi

GO_NEEDED_MAJOR=1
GO_NEEDED_MINOR=18

needs_install=true
if [[ -n "$GO_CURRENT" ]]; then
    CUR_MAJOR=$(echo "$GO_CURRENT" | cut -d'.' -f1)
    CUR_MINOR=$(echo "$GO_CURRENT" | cut -d'.' -f2)
    if [[ "$CUR_MAJOR" -gt "$GO_NEEDED_MAJOR" ]] || \
       [[ "$CUR_MAJOR" -eq "$GO_NEEDED_MAJOR" && "$CUR_MINOR" -ge "$GO_NEEDED_MINOR" ]]; then
        needs_install=false
    fi
fi

if [[ "$needs_install" == "true" ]]; then
    inf "Go belum ada atau versi terlalu lama (${GO_CURRENT:-tidak ada})"
    inf "Akan install Go ${GO_VERSION} dari sumber resmi..."
    echo ""

    # Coba apt dulu (lebih cepat, tidak perlu download besar)
    inf "Mencoba install via apt-get..."
    apt-get install -y -qq golang-go 2>/dev/null
    setup_go_path

    GO_APT_VER=""
    if command -v go &>/dev/null; then
        GO_APT_VER=$(go version 2>/dev/null | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
        APT_MAJOR=$(echo "$GO_APT_VER" | cut -d'.' -f1)
        APT_MINOR=$(echo "$GO_APT_VER" | cut -d'.' -f2)
    fi

    if [[ -n "$GO_APT_VER" ]] && \
       [[ "$APT_MAJOR" -gt "$GO_NEEDED_MAJOR" || \
          ("$APT_MAJOR" -eq "$GO_NEEDED_MAJOR" && "$APT_MINOR" -ge "$GO_NEEDED_MINOR") ]]; then
        ok "Go ${GO_APT_VER} berhasil via apt-get"
    else
        warn "apt-get menghasilkan Go ${GO_APT_VER:-tidak ada} (butuh >= ${GO_NEEDED_MAJOR}.${GO_NEEDED_MINOR})"
        inf "Menginstall Go ${GO_VERSION} dari go.dev (official binary)..."
        echo ""

        if install_go_official; then
            setup_go_path
        else
            # Fallback: coba mirror alternatif
            warn "go.dev gagal, coba mirror alternatif..."
            ALT_URL="https://golang.org/dl/${GO_TARBALL}"
            wget -q --tries=3 --timeout=120 -O /tmp/${GO_TARBALL} "$ALT_URL" 2>/dev/null || \
            curl -sL --retry 3 --max-time 120 -o /tmp/${GO_TARBALL} "$ALT_URL" 2>/dev/null

            if [[ -s /tmp/${GO_TARBALL} ]] && tar -tzf /tmp/${GO_TARBALL} &>/dev/null; then
                [[ -d "${GO_INSTALL_DIR}/go" ]] && rm -rf "${GO_INSTALL_DIR}/go"
                tar -C "${GO_INSTALL_DIR}" -xzf /tmp/${GO_TARBALL}
                rm -f /tmp/${GO_TARBALL}
                setup_go_path
                ok "Go berhasil dari mirror alternatif"
            else
                err "Semua metode install Go gagal!"
                err "Install manual: wget ${GO_URL}"
                err "Lalu: tar -C /usr/local -xzf ${GO_TARBALL}"
                exit 1
            fi
        fi
    fi
else
    ok "Go ${GO_CURRENT} sudah terinstall dan cukup baru"
    setup_go_path
fi

# Verifikasi final
if ! command -v go &>/dev/null && [[ ! -x "${GO_BIN}" ]]; then
    err "Go tidak ditemukan setelah instalasi!"
    exit 1
fi

GO_FINAL_VER=$(go version 2>/dev/null || ${GO_BIN} version 2>/dev/null)
ok "Go siap: ${GO_FINAL_VER}"

# ════════════════════════════════════════════════════════════
# STEP 3 — INSTALL ZIVPN BINARY & SERVICE
# ════════════════════════════════════════════════════════════
step "STEP 3 — Install ZiVPN Binary & Service"

BINARY_URL="https://github.com/fauzanihanipah/ziv-udp/releases/download/udp-zivpn/udp-zivpn-linux-amd64"
CONFIG_URL="https://raw.githubusercontent.com/fauzanihanipah/ziv-udp/main/config.json"
ZIVPN_BIN="/usr/local/bin/zivpn-bin"
ZIVPN_SVC="/etc/systemd/system/zivpn.service"
ZIVPN_CFG="/etc/zivpn/config.json"
ZIVPN_LOG="/etc/zivpn/zivpn.log"
ZIVPN_DOMF="/etc/zivpn/domain.conf"
ZIVPN_STRF="/etc/zivpn/store.conf"

SKIP_ZIVPN=false
if systemctl is-active --quiet zivpn 2>/dev/null; then
    warn "ZiVPN sudah berjalan."
    echo -ne "  Reinstall ZiVPN? (akan stop service) [y/N]: "
    read -r REINSTALL_ZIV
    [[ "${REINSTALL_ZIV,,}" != "y" ]] && SKIP_ZIVPN=true && \
        ok "ZiVPN dilewati — pakai instalasi yang sudah ada"
fi

if [[ "$SKIP_ZIVPN" == "false" ]]; then

    # Bersihkan instalasi lama
    systemctl stop zivpn.service 2>/dev/null
    systemctl disable zivpn.service 2>/dev/null
    rm -f "$ZIVPN_BIN" "$ZIVPN_SVC"
    rm -f /etc/zivpn/zivpn.key /etc/zivpn/zivpn.crt "$ZIVPN_CFG" "$ZIVPN_LOG"
    systemctl daemon-reload 2>/dev/null
    ok "Direktori & file lama dibersihkan"

    mkdir -p /etc/zivpn
    touch /etc/zivpn/users.db "$ZIVPN_LOG"

    # ── Input konfigurasi ────────────────────────────────────
    echo ""
    PUBLIC_IP=$(curl -s4 --max-time 8 ifconfig.me 2>/dev/null || \
                curl -s4 --max-time 8 api.ipify.org 2>/dev/null || \
                hostname -I 2>/dev/null | awk '{print $1}')
    echo -e "  ${DIM}IP Publik terdeteksi: ${CYAN}${PUBLIC_IP}${NC}"
    echo ""
    echo -ne "  ${CYAN}Domain / IP${NC} [${PUBLIC_IP}]: "; read -r INP_DOMAIN
    [[ -z "$INP_DOMAIN" ]] && INP_DOMAIN="$PUBLIC_IP"

    echo -ne "  ${CYAN}Port ZiVPN${NC}  [5667]       : "; read -r INP_PORT
    [[ -z "$INP_PORT" ]] && INP_PORT="5667"
    if ! [[ "$INP_PORT" =~ ^[0-9]+$ ]] || (( INP_PORT < 1 || INP_PORT > 65535 )); then
        warn "Port tidak valid, pakai default 5667"; INP_PORT="5667"
    fi

    echo -ne "  ${CYAN}Nama Brand${NC}  [OGH-ZIV]    : "; read -r INP_BRAND
    [[ -z "$INP_BRAND" ]] && INP_BRAND="OGH-ZIV"
    echo ""

    echo "$INP_DOMAIN" > "$ZIVPN_DOMF"
    printf "BRAND=%s\nADMIN_TG=-\n" "$INP_BRAND" > "$ZIVPN_STRF"
    ok "Konfigurasi disimpan (domain: ${INP_DOMAIN}, port: ${INP_PORT})"

    # ── Download binary ZiVPN ────────────────────────────────
    echo ""
    inf "Download ZiVPN binary..."
    echo ""
    wget --progress=bar:force:noscroll --tries=3 --timeout=180 \
         -O "$ZIVPN_BIN" "$BINARY_URL" 2>&1

    if [[ ! -s "$ZIVPN_BIN" ]]; then
        warn "wget gagal, coba curl..."
        curl -L --progress-bar --retry 3 --max-time 180 \
             -o "$ZIVPN_BIN" "$BINARY_URL"
    fi

    if [[ ! -s "$ZIVPN_BIN" ]]; then
        err "Download binary ZiVPN GAGAL!"
        warn "Download manual:"
        warn "  wget '${BINARY_URL}' -O ${ZIVPN_BIN}"
        rm -f "$ZIVPN_BIN"
        exit 1
    fi

    chmod +x "$ZIVPN_BIN"
    ok "Binary ZiVPN: ${ZIVPN_BIN} ($(du -sh "$ZIVPN_BIN" | cut -f1))"

    # ── Download config.json ─────────────────────────────────
    echo ""
    inf "Download config.json..."
    wget -q --tries=3 --timeout=30 -O "$ZIVPN_CFG" "$CONFIG_URL" 2>/dev/null || \
    curl -sL --retry 3 --max-time 30 -o "$ZIVPN_CFG" "$CONFIG_URL" 2>/dev/null

    if [[ -s "$ZIVPN_CFG" ]]; then
        python3 - << PYEOF 2>/dev/null
import json
try:
    with open('${ZIVPN_CFG}') as f: c = json.load(f)
    c['listen'] = ':${INP_PORT}'
    c['cert']   = '/etc/zivpn/zivpn.crt'
    c['key']    = '/etc/zivpn/zivpn.key'
    with open('${ZIVPN_CFG}', 'w') as f: json.dump(c, f, indent=2)
except Exception as e: print(f'Warning: {e}')
PYEOF
        ok "config.json dari GitHub (port: ${INP_PORT})"
    else
        warn "Tidak bisa download config.json — membuat manual..."
        cat > "$ZIVPN_CFG" << CFGJSON
{
  "listen": ":${INP_PORT}",
  "cert": "/etc/zivpn/zivpn.crt",
  "key": "/etc/zivpn/zivpn.key",
  "obfs": "zivpn",
  "auth": {
    "mode": "passwords",
    "config": []
  }
}
CFGJSON
        ok "config.json dibuat manual (port: ${INP_PORT})"
    fi

    # ── Generate SSL Certificate ─────────────────────────────
    echo ""
    inf "Generate SSL Certificate..."
    openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 \
        -subj "/C=ID/ST=Indonesia/L=Jakarta/O=${INP_BRAND}/CN=${INP_DOMAIN}" \
        -keyout /etc/zivpn/zivpn.key \
        -out    /etc/zivpn/zivpn.crt 2>/dev/null

    if [[ $? -ne 0 ]]; then
        warn "RSA-4096 gagal, coba EC P-256..."
        openssl req -x509 -nodes -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
            -days 3650 -subj "/CN=${INP_DOMAIN}" \
            -keyout /etc/zivpn/zivpn.key \
            -out    /etc/zivpn/zivpn.crt 2>/dev/null || {
                err "SSL generation gagal!"; exit 1
            }
        ok "SSL EC P-256 (10 tahun) dibuat"
    else
        ok "SSL RSA-4096 (1 tahun) dibuat"
    fi

    # ── Optimasi kernel UDP ──────────────────────────────────
    sysctl -w net.core.rmem_max=16777216 &>/dev/null
    sysctl -w net.core.wmem_max=16777216 &>/dev/null
    grep -q 'rmem_max' /etc/sysctl.conf 2>/dev/null || \
        printf "\nnet.core.rmem_max=16777216\nnet.core.wmem_max=16777216\n" >> /etc/sysctl.conf
    ok "Kernel buffer UDP dioptimasi (16 MB)"

    # ── Systemd service ZiVPN ────────────────────────────────
    echo ""
    inf "Membuat systemd service ZiVPN..."
    cat > "$ZIVPN_SVC" << SVCEOF
[Unit]
Description=ZiVPN UDP Server (OGH-ZIV)
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn
ExecStart=${ZIVPN_BIN} server -c ${ZIVPN_CFG}
Restart=always
RestartSec=3
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
NoNewPrivileges=true
LimitNOFILE=1048576
StandardOutput=append:${ZIVPN_LOG}
StandardError=append:${ZIVPN_LOG}

[Install]
WantedBy=multi-user.target
SVCEOF
    ok "Service file ZiVPN dibuat"

    # ── IPTables forwarding 6000-19999 → port VPN ────────────
    echo ""
    inf "Setup iptables UDP forwarding (6000-19999 → ${INP_PORT})..."
    IFACE=$(ip -4 route ls 2>/dev/null | grep default | grep -oP '(?<=dev )(\S+)' | head -1)
    [[ -z "$IFACE" ]] && IFACE=$(ip link show 2>/dev/null | \
        awk -F: '$0 !~ "lo|vir|^[^0-9]"{print $2;exit}' | tr -d ' ')
    [[ -z "$IFACE" ]] && IFACE="eth0"

    # Bersihkan rules lama
    while iptables -t nat -D PREROUTING \
          -i "$IFACE" -p udp --dport 6000:19999 \
          -j DNAT --to-destination ":${INP_PORT}" 2>/dev/null; do :; done

    # Tambah rules baru
    iptables -t nat -A PREROUTING \
        -i "$IFACE" -p udp --dport 6000:19999 \
        -j DNAT --to-destination ":${INP_PORT}" 2>/dev/null
    iptables -I INPUT -p udp --dport "${INP_PORT}" -j ACCEPT 2>/dev/null
    iptables -A FORWARD -p udp -d 127.0.0.1 --dport "${INP_PORT}" -j ACCEPT 2>/dev/null
    iptables -t nat -A POSTROUTING -s 127.0.0.1/32 -o "$IFACE" -j MASQUERADE 2>/dev/null

    # UFW jika ada
    if command -v ufw &>/dev/null; then
        ufw allow 6000:19999/udp &>/dev/null
        ufw allow "${INP_PORT}/udp" &>/dev/null
        ok "UFW: port UDP dibuka"
    fi

    # Simpan rules permanen
    netfilter-persistent save &>/dev/null
    ok "IPTables rules disimpan permanen"
    ok "UDP forwarding: 6000-19999 → ${INP_PORT} via ${IFACE}"

    # ── Start ZiVPN ──────────────────────────────────────────
    echo ""
    inf "Aktifkan & jalankan ZiVPN..."
    systemctl daemon-reload
    systemctl enable zivpn.service &>/dev/null
    systemctl start zivpn.service
    sleep 2

    if systemctl is-active --quiet zivpn; then
        ok "✅ Service ZiVPN AKTIF & berjalan!"
    else
        warn "ZiVPN gagal start — cek log:"
        journalctl -u zivpn -n 10 --no-pager 2>/dev/null | sed 's/^/      /'
        echo -ne "  Lanjutkan install bot? [Y/n]: "; read -r CONT_ANS
        [[ "${CONT_ANS,,}" == "n" ]] && exit 1
    fi

    echo ""
    echo -e "  ${GREEN}┌──────────────────────────────────────────┐${NC}"
    echo -e "  ${GREEN}│${NC}  ${BOLD}ZiVPN Berhasil Diinstall!${NC}               ${GREEN}│${NC}"
    echo -e "  ${GREEN}├──────────────┬───────────────────────────┤${NC}"
    printf  "  ${GREEN}│${NC} Domain/IP   ${GREEN}│${NC}  %-25s  ${GREEN}│${NC}\n" "$INP_DOMAIN"
    printf  "  ${GREEN}│${NC} Port        ${GREEN}│${NC}  %-25s  ${GREEN}│${NC}\n" "$INP_PORT"
    printf  "  ${GREEN}│${NC} Forwarding  ${GREEN}│${NC}  %-25s  ${GREEN}│${NC}\n" "UDP 6000-19999 → ${INP_PORT}"
    printf  "  ${GREEN}│${NC} Interface   ${GREEN}│${NC}  %-25s  ${GREEN}│${NC}\n" "$IFACE"
    echo -e "  ${GREEN}└──────────────┴───────────────────────────┘${NC}"

fi  # END SKIP_ZIVPN

# ════════════════════════════════════════════════════════════
# STEP 4 — COMPILE & INSTALL TELEGRAM BOT
# ════════════════════════════════════════════════════════════
step "STEP 4 — Compile & Install Telegram Bot"

inf "Membuat direktori /opt/zivpn-bot..."
mkdir -p /opt/zivpn-bot

inf "Salin source code..."
cp "${BOT_SRC}" /opt/zivpn-bot/main.go
ok "Source disalin ke /opt/zivpn-bot/main.go"

cd /opt/zivpn-bot

# Init Go module
inf "Inisialisasi Go module..."
if [[ ! -f go.mod ]]; then
    go mod init zivpn-bot 2>&1 | sed 's/^/    /'
fi
ok "Go module siap"

# Compile
inf "Kompilasi bot (mungkin butuh beberapa detik)..."
echo ""
BUILD_LOG=$(go build -v -o /usr/local/bin/zivpn-bot main.go 2>&1)
BUILD_EXIT=$?
echo ""

if [[ $BUILD_EXIT -ne 0 ]]; then
    err "Kompilasi GAGAL!"
    echo ""
    echo -e "  ${RED}Detail error:${NC}"
    echo "$BUILD_LOG" | sed 's/^/    /'
    echo ""
    err "Cek file bot-zivpn.go dan coba lagi."
    exit 1
fi

chmod +x /usr/local/bin/zivpn-bot
BINARY_SIZE=$(du -sh /usr/local/bin/zivpn-bot 2>/dev/null | cut -f1)
ok "Kompilasi berhasil! Binary: /usr/local/bin/zivpn-bot (${BINARY_SIZE})"

# ── Systemd service bot ─────────────────────────────────────

# Stop dulu jika sudah berjalan
if systemctl is-active --quiet zivpn-bot 2>/dev/null; then
    inf "Menghentikan service lama..."
    systemctl stop zivpn-bot
fi

inf "Membuat service file..."
cat > /etc/systemd/system/zivpn-bot.service << 'SVCEOF'
[Unit]
Description=OGH-ZIV Telegram Bot
Documentation=https://github.com/fauzanihanipah
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn
ExecStart=/usr/local/bin/zivpn-bot
Restart=always
RestartSec=5
StartLimitInterval=60
StartLimitBurst=5
StandardOutput=append:/etc/zivpn/bot.log
StandardError=append:/etc/zivpn/bot.log
Environment=HOME=/root

[Install]
WantedBy=multi-user.target
SVCEOF

ok "Service file dibuat"

inf "Reload systemd daemon..."
systemctl daemon-reload

inf "Enable service (auto-start)..."
systemctl enable zivpn-bot.service &>/dev/null
ok "Service di-enable"

inf "Menjalankan service..."
systemctl start zivpn-bot.service
sleep 3

if systemctl is-active --quiet zivpn-bot; then
    ok "Service zivpn-bot AKTIF & berjalan!"
else
    err "Service gagal start!"
    echo ""
    warn "Cek log berikut:"
    journalctl -u zivpn-bot -n 20 --no-pager 2>/dev/null | sed 's/^/    /'
    echo ""
    warn "Kemungkinan penyebab:"
    warn "- BOT_TOKEN tidak valid"
    warn "- Tidak ada koneksi internet ke api.telegram.org"
    warn "- Binary rusak (coba compile ulang)"
    echo ""
    warn "Coba jalankan manual untuk debug:"
    warn "  /usr/local/bin/zivpn-bot"
fi

# ════════════════════════════════════════════════════════════
# STEP 5 — SETUP ADMIN & PAKET
# ════════════════════════════════════════════════════════════
step "STEP 5 — Setup Admin & Paket"

ADMIN_DB="/etc/zivpn/admins.db"
CHAT_ID=$(grep "^CHAT_ID=" /etc/zivpn/bot.conf 2>/dev/null | cut -d'=' -f2 | tr -d ' ')

if [[ -n "$CHAT_ID" && ! -f "$ADMIN_DB" ]]; then
    inf "Menambahkan Chat ID dari bot.conf sebagai admin awal..."
    echo "# Admin Telegram IDs — satu ID per baris" > "$ADMIN_DB"
    echo "$CHAT_ID" >> "$ADMIN_DB"
    ok "Admin awal: ${CHAT_ID}"
elif [[ -f "$ADMIN_DB" ]]; then
    ADMIN_COUNT=$(grep -c '^[0-9]' "$ADMIN_DB" 2>/dev/null || echo 0)
    ok "File admin sudah ada ($ADMIN_COUNT admin terdaftar)"
else
    warn "Belum ada admin! Tambahkan ID Telegram kamu ke:"
    warn "  /etc/zivpn/admins.db"
    echo ""
    echo -ne "  Masukkan Telegram ID admin kamu (Enter untuk skip): "
    read -r MANUAL_ADMIN_ID
    if [[ -n "$MANUAL_ADMIN_ID" && "$MANUAL_ADMIN_ID" =~ ^[0-9]+$ ]]; then
        echo "# Admin Telegram IDs — satu ID per baris" > "$ADMIN_DB"
        echo "$MANUAL_ADMIN_ID" >> "$ADMIN_DB"
        ok "Admin ditambahkan: ${MANUAL_ADMIN_ID}"
        systemctl restart zivpn-bot
    else
        warn "Dilewati. Tambahkan ID admin manual nanti."
    fi
fi

# Init paket default jika belum ada
PAKET_DB="/etc/zivpn/paket.db"
if [[ ! -f "$PAKET_DB" ]]; then
    inf "Membuat paket default..."
    cat > "$PAKET_DB" << 'PAKETEOF'
30 Hari|30|10000|0|2
PAKETEOF
    ok "Paket default dibuat (30 Hari — Rp 10.000)"
else
    ok "Paket sudah ada"
fi

# ════════════════════════════════════════════════════════════
# SELESAI
# ════════════════════════════════════════════════════════════
echo ""
ZIVPN_STATUS="❌ STOPPED"
BOT_STATUS="❌ STOPPED"
systemctl is-active --quiet zivpn     2>/dev/null && ZIVPN_STATUS="✅ RUNNING"
systemctl is-active --quiet zivpn-bot 2>/dev/null && BOT_STATUS="✅ RUNNING"

echo -e "  ${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "  ${CYAN}║${NC}  ${GREEN}${BOLD}  ✦  OGH-ZIV FULL INSTALL SELESAI!  ✦${NC}               ${CYAN}║${NC}"
echo -e "  ${CYAN}╠══════════════════════════════════════════════════════════╣${NC}"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "Go Version  : $(go version 2>/dev/null | awk '{print $3}')"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "ZiVPN       : ${ZIVPN_STATUS}"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "Telegram Bot: ${BOT_STATUS}"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "Bot Binary  : /usr/local/bin/zivpn-bot"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "ZiVPN Log   : /etc/zivpn/zivpn.log"
printf  "  ${CYAN}║${NC}  %-54s  ${CYAN}║${NC}\n" "Bot Log     : /etc/zivpn/bot.log"
echo -e "  ${CYAN}╠══════════════════════════════════════════════════════════╣${NC}"
echo -e "  ${CYAN}║${NC}  ${YELLOW}Perintah berguna:${NC}                                      ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  systemctl status  zivpn       — status ZiVPN          ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  systemctl restart zivpn       — restart ZiVPN         ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  systemctl status  zivpn-bot   — status bot            ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  systemctl restart zivpn-bot   — restart bot           ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  tail -f /etc/zivpn/zivpn.log  — log ZiVPN live        ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  tail -f /etc/zivpn/bot.log    — log bot live          ${CYAN}║${NC}"
echo -e "  ${CYAN}╠══════════════════════════════════════════════════════════╣${NC}"
echo -e "  ${CYAN}║${NC}  ${YELLOW}File penting:${NC}                                          ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  /etc/zivpn/admins.db  — daftar Telegram ID admin      ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  /etc/zivpn/paket.db   — daftar paket & harga          ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  /etc/zivpn/bot.conf   — token & config bot            ${CYAN}║${NC}"
echo -e "  ${CYAN}║${NC}  /etc/zivpn/orders.db  — database order masuk          ${CYAN}║${NC}"
echo -e "  ${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${DIM}Cara cek Telegram ID: chat @userinfobot di Telegram${NC}"
echo ""
