# CashMate API

REST API cash management UMKM. Stack: Go, Gin, GORM, MySQL, JWT.

## Scope

- Tenant: `Business` = satu UMKM.
- User: satu Business, role `OWNER` atau `STAFF`.
- Currency: `IDR`.
- Amount/balance: integer rupiah, bukan float.
- Database production: eksternal, bukan container Docker.

## Jalankan lokal

Prasyarat: Docker Compose dan Go.

```bash
docker compose -f docker-compose.dev.yml up -d --build
curl http://localhost:8096/api/health
```

Development memakai MySQL lokal di port `3307` dan API di `8096`.

Tanpa Docker:

```bash
cp .env.development.example .env
go run .
```

`AUTO_MIGRATE=true` hanya untuk database development kosong.

## Environment

Wajib di production:

```dotenv
APP_PORT=8096
APP_TIMEZONE=Asia/Jakarta
AUTO_MIGRATE=false
DB_HOST=<external-mysql-host>
DB_PORT=3306
DB_USER=<db-user>
DB_PASSWORD=<db-password>
DB_NAME=<db-name>
JWT_SECRET=<random-secret-min-32-chars>
JWT_REFRESH_SECRET=<different-random-secret>
CORS_ORIGINS=https://<web-domain>
```

## Aturan API

- `POST /api/auth/register` selalu membuat Business baru, Owner, dan wallet `Cash`.
- Staff hanya dapat dibuat Owner melalui `/api/staff`.
- Semua resource otomatis dibatasi ke Business user yang login.
- `business_id` dari body/query/path tidak digunakan sebagai authorization.
- Staff tidak dapat membuka dashboard/report, melihat total balance, backdate, edit, atau void transaksi.
- Staff hanya melihat transaksi miliknya pada hari bisnis saat ini.
- Wallet balance hanya berubah melalui transaksi; wallet CRUD tidak boleh mengubah balance.
- Transaksi void tidak masuk dashboard/report.

## Endpoint

| Method | Endpoint | Akses |
|---|---|---|
| POST | `/api/auth/register` | Public, membuat Owner |
| POST | `/api/auth/login` | Public |
| POST | `/api/auth/refresh` | Public |
| POST | `/api/auth/logout` | Authenticated |
| GET | `/api/auth/me` | Authenticated |
| GET/POST | `/api/staff` | Owner |
| DELETE | `/api/staff/:id` | Owner |
| GET | `/api/wallets[/:id]` | Owner/Staff; Staff tanpa balance |
| POST/PUT/DELETE | `/api/wallets[/:id]` | Owner |
| POST | `/api/wallets/:id/restore` | Owner |
| GET | `/api/categories` | Owner/Staff |
| POST/PUT/DELETE | `/api/categories[/:id]` | Owner |
| POST | `/api/categories/:id/restore` | Owner |
| GET/POST | `/api/transactions` | Owner/Staff create |
| PUT/DELETE | `/api/transactions/:id` | Owner |
| POST | `/api/transactions/:id/restore` | Owner |
| GET | `/api/dashboard/summary` | Owner |
| GET | `/api/reports/monthly?year=2026` | Owner |

Semua endpoint selain auth dan health memakai:

```http
Authorization: Bearer <access_token>
```

## Payload utama

Registration:

```json
{
  "business_name": "Demo Warung",
  "name": "Owner Demo",
  "email": "owner@example.com",
  "password": "password123"
}
```

Transaction:

```json
{
  "wallet_id": 1,
  "category_id": 2,
  "amount": 25000,
  "type": "expense",
  "description": "Pembelian bahan",
  "date": "2026-09-15"
}
```

`amount` harus `> 0`. `type` harus `income` atau `expense`. `date` hanya boleh dikirim Owner.

## Response contract

Success:

```json
{ "message": "...", "data": {}, "meta": {} }
```

`meta` hanya ada pada response pagination. Error:

```json
{ "message": "...", "errors": {} }
```

Cross-tenant resource: `404`. Role tidak diizinkan: `403`.

## Migrasi production

Script berikut meng-upgrade schema lama yang masih memiliki `wallets.user_id` dan `categories.user_id`.
Jalankan saat API lama dihentikan, pada backup/clone terlebih dahulu.

1. Backup:

```bash
mysqldump --single-transaction --routines --triggers \
  -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p \
  "$DB_NAME" > cashmate-before-migration.sql
```

2. Preflight. Hentikan proses jika ada owner aktif yang perlu dipetakan ulang atau nominal pecahan rupiah:

```sql
SELECT role, COUNT(*) FROM users WHERE deleted_at IS NULL GROUP BY role;
SELECT id, balance FROM wallets WHERE balance <> TRUNCATE(balance, 0);
SELECT id, amount FROM transactions WHERE amount <> TRUNCATE(amount, 0);
```

3. Jalankan berurutan:

```bash
mysql -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p "$DB_NAME" \
  < migrations/001_create_business_tenancy.sql
mysql -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p "$DB_NAME" \
  < migrations/002_backfill_legacy_business.sql
mysql -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p "$DB_NAME" \
  < migrations/003_convert_money_and_constraints.sql
```

`002` memetakan data lama ke satu Business `CashMate Legacy` dan membuat adjustment transaction bila cached balance berbeda dari ledger.

4. Deploy API baru dengan `AUTO_MIGRATE=false`, lalu smoke test registration, login, staff, transaction, dashboard, dan tenant isolation.
5. Setelah smoke test berhasil, hapus ownership lama:

```bash
mysql -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p "$DB_NAME" \
  < migrations/004_remove_legacy_ownership.sql
```

Migration tidak memiliki down script. Rollback dilakukan dengan restore backup. Jangan menjalankan migration versioned ulang tanpa memeriksa schema.

## Deploy VPS

`docker-compose.yml` production hanya menjalankan API, Nginx, dan Certbot. Tidak ada image/service MySQL.

```bash
./scripts/deploy.sh
```

Pastikan `.env` production sudah ada dan `AUTO_MIGRATE=false`.

## Test

```bash
go test ./...
go vet ./...
```

Integration test membutuhkan database lokal dengan nama mengandung `_test`:

```bash
CASHMATE_TEST_DSN='cashmate:cashmate@tcp(127.0.0.1:3307)/cashmate_test?charset=utf8mb4&parseTime=True&loc=Asia%2FJakarta' \
  go test ./integration -v
```

Postman collection tunggal:

`postman/cashmate_postman_collection.json`
