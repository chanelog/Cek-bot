# 🤖 OGH-ZIV Telegram Bot — Panduan Lengkap

## 📋 Deskripsi
Bot Telegram publik untuk penjualan VPN otomatis dengan:
- ✅ Pembayaran via **DANA 083113931971** a/n Fauzani Hanifah
- ✅ Bayar dulu → Upload bukti → Admin konfirmasi → Akun otomatis dibuat
- ✅ Admin bisa buat akun **gratis tanpa bayar**
- ✅ Terhubung langsung ke sistem `ogh-ziv` (shared database `/etc/zivpn/`)

---

## 🚀 Cara Install

### Prasyarat
1. Panel `ogh-ziv` sudah terinstall
2. Telegram Bot sudah di-setup via menu panel (Telegram Bot → Setup)
3. File `/etc/zivpn/bot.conf` sudah ada dan berisi `BOT_TOKEN`

### Langkah Install
```bash
# Upload kedua file ke VPS, lalu:
chmod +x install-bot.sh
bash install-bot.sh
```

### Jika Go belum terinstall (manual)
```bash
# Ubuntu/Debian
apt-get install golang-go

# Atau download langsung
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Compile Manual
```bash
mkdir -p /opt/zivpn-bot
cp bot-zivpn.go /opt/zivpn-bot/main.go
cd /opt/zivpn-bot
go mod init zivpn-bot
go build -o /usr/local/bin/zivpn-bot main.go
```

---

## ⚙️ Konfigurasi

### 1. Tambah Admin
Edit file `/etc/zivpn/admins.db`:
```
# Admin IDs — satu Telegram ID per baris
123456789
987654321
```

Cara cek Telegram ID kamu: chat `@userinfobot` di Telegram.

Atau via bot (sudah jadi admin):
```
/setadmin 123456789
```

### 2. Kelola Paket
File: `/etc/zivpn/paket.db`
Format: `Nama|Hari|Harga|KuotaGB|MaxLogin`
```
1 Hari|1|10000|0|2
7 Hari|7|10000|0|2
30 Hari|30|10000|0|2
90 Hari|90|10000|0|3
```
- KuotaGB = 0 berarti Unlimited

Via bot:
```
/newpaket 30 Hari VIP|30|10000|0|3
/delpaket 1 Hari
/setpaket
```

---

## 👥 Cara Pakai (User Publik)

1. Start bot → `/start`
2. Tekan **🛒 Beli VPN**
3. Pilih nomor paket
4. Masukkan username (huruf kecil/angka, 4-20 karakter)
5. Masukkan password (min 6 karakter, atau `auto`)
6. Bot tampilkan info pembayaran DANA
7. Transfer sesuai nominal
8. Upload screenshot bukti transfer
9. Tunggu konfirmasi admin
10. Akun langsung aktif & dikirim ke chat!

---

## 👑 Cara Pakai (Admin)

### Lihat Order Pending
```
/listorder
```

### Konfirmasi Order (Akun Otomatis Dibuat)
```
/confirm ORDERID username password hari kuota maxlogin
```
Contoh:
```
/confirm ZIV-01021504051234 john123 mypass123 30 0 2
```
> Bot akan otomatis membuat akun di sistem dan mengirim detail ke user.

### Tolak Order
```
/reject ORDERID alasan
```

### Buat Akun Gratis (Tanpa Bayar)
```
/createakun username password hari kuota maxlogin [note]
```
Contoh:
```
/createakun admin123 securepass 365 0 5 VIP Reseller
/createakun tester pass123 7 1 2
```

### Broadcast ke Semua User
```
/broadcast
```
Lalu ketik pesan, tekan Enter.

---

## 📁 File Database

| File | Keterangan |
|------|-----------|
| `/etc/zivpn/users.db` | Database akun VPN (shared dengan panel) |
| `/etc/zivpn/orders.db` | Database order bot |
| `/etc/zivpn/paket.db` | Daftar paket & harga |
| `/etc/zivpn/admins.db` | Daftar Telegram ID admin |
| `/etc/zivpn/bot.conf` | Token bot & Chat ID |
| `/etc/zivpn/maxlogin.db` | Batas login per akun |

---

## 🔧 Manajemen Service

```bash
# Status
systemctl status zivpn-bot

# Restart
systemctl restart zivpn-bot

# Stop
systemctl stop zivpn-bot

# Lihat log real-time
tail -f /etc/zivpn/bot.log

# Log systemd
journalctl -u zivpn-bot -f
```

---

## 📊 Alur Pembelian

```
User /start
    │
    ▼
Pilih Paket
    │
    ▼
Input Username & Password
    │
    ▼
Bot Buat Order ID + Tampilkan Info DANA
    │
    ▼
User Transfer ke DANA 083113931971
    │
    ▼
User Upload Screenshot Bukti
    │
    ▼
Admin Terima Notifikasi + Foto Bukti
    │
    ▼
Admin /confirm → Akun Otomatis Dibuat
    │
    ▼
User Terima Detail Akun VPN ✅
```

---

## ⚠️ Catatan Penting

- Bot menggunakan **polling** (bukan webhook), cocok untuk VPS
- Database shared dengan panel `ogh-ziv`, perubahan sinkron otomatis
- Saat admin `/confirm`, ZiVPN service di-restart otomatis
- Simpan backup `/etc/zivpn/` secara berkala
