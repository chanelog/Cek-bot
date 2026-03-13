# OGH-ZIV Telegram Bot — Auto Create Akun VPN

Bot Telegram otomatis untuk membuat akun VPN premium setelah pembayaran via **DANA** diverifikasi dari screenshot.

Terintegrasi penuh dengan **OGH-ZIV Premium Panel**.

---

## ✨ Fitur

- 🛒 Pilih paket VPN langsung di Telegram
- 📸 Kirim screenshot DANA → akun langsung dibuat otomatis
- 🤖 OCR otomatis baca screenshot (Tesseract)
- 👮 Fallback verifikasi manual oleh admin jika OCR tidak tersedia
- 🔒 Hanya admin yang bisa buat akun gratis
- 📋 Tampilkan: IP, Username, Password, Expired, Max Login
- 💬 Cantumkan kontak admin untuk keluhan
- 🔔 Notifikasi ke admin setiap ada pesanan baru

---

## 📦 Instalasi

### Prasyarat
- VPS Debian / Ubuntu
- OGH-ZIV Panel sudah terinstall
- Token Bot dari [@BotFather](https://t.me/BotFather)
- Chat ID Telegram admin (bisa dari [@userinfobot](https://t.me/userinfobot))

### Install Otomatis

```bash
bash <(curl -Ls https://raw.githubusercontent.com/chanelog/Cek-bot/main/install-tgbot.sh)
```

### Install Manual

```bash
# 1. Clone repo
git clone https://github.com/chanelog/Cek-bot
cd Cek-bot

# 2. Jalankan installer
chmod +x install-tgbot.sh
bash install-tgbot.sh
```

---

## ⚙️ Konfigurasi

File konfigurasi: `/etc/zivpn/bot_store.conf`

```ini
BOT_TOKEN=1234567890:AAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
ADMIN_IDS=123456789
DANA_NUMBER=081234567890
DANA_NAME=Nama Pemilik Dana
BRAND=OGH-ZIV
ADMIN_TG=@namaadmin
```

| Parameter     | Keterangan                          |
|--------------|--------------------------------------|
| `BOT_TOKEN`  | Token bot dari @BotFather            |
| `ADMIN_IDS`  | Chat ID admin (pisah koma jika >1)  |
| `DANA_NUMBER`| Nomor DANA penerima pembayaran       |
| `DANA_NAME`  | Nama pemilik akun DANA               |
| `BRAND`      | Nama brand yang tampil di bot        |
| `ADMIN_TG`   | Username Telegram admin untuk keluhan|

---

## 💰 Paket (Edit di `zivpn_bot.py`)

| Paket   | Harga     | Durasi  | Kuota    | Max Login |
|---------|-----------|---------|----------|-----------|
| 1 Hari  | Rp 2.000  | 1 hari  | 1 GB     | 1 device  |
| 7 Hari  | Rp 10.000 | 7 hari  | 5 GB     | 2 device  |
| 30 Hari | Rp 30.000 | 30 hari | 20 GB    | 2 device  |
| 90 Hari | Rp 80.000 | 90 hari | Unlimited| 3 device  |

---

## 🤖 Cara Kerja Bot

```
User /start
  ↓
Pilih Paket
  ↓
Bot tampilkan info pembayaran DANA
  ↓
User transfer & kirim screenshot
  ↓
Bot cek screenshot via OCR (Tesseract)
  ├── Cocok → Buat akun otomatis → Kirim info akun ke user
  └── Tidak cocok / OCR error → Forward ke admin untuk verifikasi manual
        ↓
      Admin /konfirm_<user_id> → Akun dibuat & dikirim ke user
      Admin /tolak_<user_id>   → User diberi notifikasi penolakan
```

---

## 📱 Informasi Akun yang Dikirim ke Pembeli

```
🎉 OGH-ZIV — Akun VPN Premium
━━━━━━━━━━━━━━━━━━━━━━━
🖥 IP Publik  : 123.456.789.0
🌐 Host      : domain.kamu.com
🔌 Port      : 5667
📡 Obfs      : zivpn
━━━━━━━━━━━━━━━━━━━━━━━
👤 Username  : ziv12345
🔑 Password  : Ab3xK9mN2pQr
━━━━━━━━━━━━━━━━━━━━━━━
📦 Kuota     : 20 GB
🔒 Max Login : 2 device
📅 Expired   : 2025-08-01 (30 hari lagi)
━━━━━━━━━━━━━━━━━━━━━━━
💬 Keluhan & bantuan: @namaadmin
```

---

## 🛠️ Perintah Manajemen

```bash
# Status bot
systemctl status zivpn-tgbot

# Restart bot
systemctl restart zivpn-tgbot

# Lihat log real-time
journalctl -u zivpn-tgbot -f

# Stop bot
systemctl stop zivpn-tgbot
```

---

## 👮 Admin Commands (via Telegram)

| Command | Fungsi |
|---------|--------|
| `/start` | Menu utama |
| Tombol **⚙️ Admin Panel** | Akses panel admin |
| **➕ Buat Akun Gratis** | Buat akun tanpa bayar (admin only) |
| **📋 List Akun** | Lihat semua akun |
| **📊 Statistik** | Info server & jumlah akun |
| `/konfirm_<id>` | Konfirmasi pembayaran manual |
| `/tolak_<id>` | Tolak pembayaran |

---

## 📁 Struktur File

```
/etc/zivpn/
├── bot_store.conf    ← Konfigurasi bot & DANA
├── bot.conf          ← Token bot (dari ogh-ziv.sh)
├── users.db          ← Database akun VPN
├── domain.conf       ← Domain/IP server
├── config.json       ← Konfigurasi ZiVPN
└── maxlogin.db       ← Data max login per user

/usr/local/bin/
└── zivpn-tgbot.py    ← Script bot utama

/etc/systemd/system/
└── zivpn-tgbot.service ← Systemd service
```

---

## 🔗 Terkait

- [OGH-ZIV Panel](https://github.com/chanelog/Socks)
- [ZiVPN Binary](https://github.com/fauzanihanipah/ziv-udp)

---

*OGH-ZIV Team — Premium VPN Management*
