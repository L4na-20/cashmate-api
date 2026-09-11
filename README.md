# ======================
#  CASHMATE API - GUIDE
#  Backend REST API Golang
# ======================

## 🧰 Tech Stack
- **Go** (Gin Gonic untuk HTTP + routing)
- **GORM** (ORM untuk MySQL)
- **MySQL** sebagai database

## 📁 Struktur Folder

```
API/
├── go.mod                      # Dependensi Go
├── .env.example                # Template konfigurasi
├── main.go                     # Entry point aplikasi
├── config/
│   └── database.go             # Load .env + koneksi GORM ke MySQL
├── models/
│   ├── category.go             # Model tabel categories
│   └── transaction.go          # Model tabel transactions (+ relasi ke kategori)
├── controllers/
│   ├── category_controller.go  # GET/POST /api/categories
│   ├── transaction_controller.go
│   ├── dashboard_controller.go # /api/dashboard/summary
│   └── report_controller.go    # /api/reports/monthly
├── middleware/
│   └── cors.go                 # Middleware CORS
└── routes/
    └── routes.go               # Registrasi semua route
```

## 🚀 Cara Menjalankan

### 1. Prasyarat
- Go 1.21+ ([download](https://go.dev/dl/))
- MySQL aktif (XAMPP / Laragon / Docker)

### 2. Persiapan Database
```sql
CREATE DATABASE cashmate CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 3. Konfigurasi
```bash
# dari folder API/
cp .env.example .env
```

Lalu sesuaikan isi `.env` (DB_USER, DB_PASSWORD, dst). Tabel `categories` dan
`transactions` dibuat **otomatis** saat server pertama kali dijalankan (AutoMigrate).

### 4. Install dependensi & jalankan
```bash
go mod tidy
go run main.go
```

Server berjalan di **http://localhost:8080**. Cek dengan:

```bash
curl http://localhost:8080/api/health
# => {"status":"ok","service":"cashmate-api"}
```

## 🔌 Daftar Endpoint

| Method | URL                              | Deskripsi                                        |
|--------|----------------------------------|--------------------------------------------------|
| GET    | `/api/health`                    | Health check                                     |
| GET    | `/api/categories`                | List kategori (`?type=income` untuk filter)      |
| POST   | `/api/categories`                | Tambah kategori                                  |
| POST   | `/api/transactions`              | Catat Cash In/Out baru                           |
| GET    | `/api/transactions`              | Histori + filter + pagination                    |
| GET    | `/api/dashboard/summary`         | Total In, Total Out, Sisa Saldo, transaksi terakhir |
| GET    | `/api/reports/monthly`           | Rekap pemasukan/pengeluaran per bulan            |

### Contoh Request

**POST `/api/categories`**
```json
{ "name": "Penjualan", "type": "income" }
```

**POST `/api/transactions`**
```json
{
  "amount": 150000,
  "type": "income",
  "category_id": 1,
  "description": "Penjualan produk A",
  "date": "2026-09-08"
}
```
> `type` harus cocok dengan `type` kategori. `date` opsional (format `YYYY-MM-DD`).

**GET `/api/transactions`** — filter & pagination
```
/api/transactions
/api/transactions?from_date=2026-09-01&to_date=2026-09-30
/api/transactions?category_id=2&type=expense
/api/transactions?page=2&per_page=20
```

## 🌐 CORS
CORS sudah dikonfigurasi di `middleware/cors.go`. Default origin yang diizinkan:
- `http://localhost:8000` (Laravel)
- `http://127.0.0.1:8000`
- `http://localhost:3000`

Tambahkan origin frontend lain pada daftar `AllowOrigins` bila perlu.

## 🧪 Uji Cepat Semua Endpoint
```bash
# Tambah kategori
curl -X POST http://localhost:8080/api/categories -H "Content-Type: application/json" -d "{\"name\":\"Penjualan\",\"type\":\"income\"}"

# Catat transaksi
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":100000,\"type\":\"income\",\"category_id\":1,\"description\":\"Test\"}"

# Ringkasan dashboard
curl http://localhost:8080/api/dashboard/summary

# Rekap bulanan
curl "http://localhost:8080/api/reports/monthly?year=2026"
```