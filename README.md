# CashMate API

REST API CashMate untuk cash management UMKM. API memakai Gin, GORM, MySQL,
JWT access/refresh token, dan tenant isolation berbasis Business.

## Development cepat

Prasyarat: Docker Compose.

```bash
docker compose -f docker-compose.dev.yml up -d --build
curl http://localhost:8096/api/health
```

Compose development menjalankan MySQL terisolasi pada port host `3307` dan API
pada `8096`. Database production tidak pernah dipakai oleh file ini.

Untuk menjalankan API di luar Docker:

```bash
cp .env.development.example .env
go run .
```

`AUTO_MIGRATE=true` hanya untuk database development kosong. Production harus
menjalankan SQL berurutan di `migrations/` setelah backup dan preflight.

## Model tenant

```text
Business
├── Owner + Staff
├── Wallets
├── Categories
└── Transactions
```

Public `POST /api/auth/register` selalu membuat Business baru, Owner, dan wallet
`Cash` dalam satu database transaction. Staff hanya dibuat oleh Owner melalui
`POST /api/staff`.

Role API adalah `OWNER` dan `STAFF`. Semua resource di-scope dari Business yang
dimuat server melalui JWT user aktif; `business_id` dari client tidak pernah
dipakai sebagai otorisasi.

## Endpoint MVP

| Method | Endpoint | Akses |
|---|---|---|
| POST | `/api/auth/register` | Public Owner registration |
| POST | `/api/auth/login` | Public |
| POST | `/api/auth/refresh` | Public |
| POST | `/api/auth/logout` | Authenticated |
| GET | `/api/auth/me` | Authenticated |
| GET/POST/DELETE | `/api/staff[/:id]` | Owner |
| GET | `/api/wallets[/:id]` | Owner/Staff; Staff tanpa balance |
| POST/PUT/DELETE | `/api/wallets[/:id]` | Owner |
| POST | `/api/wallets/:id/restore` | Owner |
| GET | `/api/categories` | Owner/Staff |
| POST/PUT/DELETE | `/api/categories[/:id]` | Owner |
| POST | `/api/categories/:id/restore` | Owner |
| GET/POST | `/api/transactions` | Owner/Staff create; Staff hanya histori sendiri hari ini |
| PUT/DELETE | `/api/transactions/:id` | Owner |
| POST | `/api/transactions/:id/restore` | Owner |
| GET | `/api/dashboard/summary` | Owner |
| GET | `/api/reports/monthly?year=2026` | Owner |

Success response memakai:

```json
{ "message": "...", "data": {} }
```

List pagination memakai `data` dan `meta`. Error memakai `message` dan optional
`errors`. Resource tenant lain dikembalikan sebagai `404`; role yang tidak
berhak dikembalikan sebagai `403`.

## Money, date, dan balance

- Currency MVP selalu `IDR`.
- Amount dan wallet balance adalah integer whole rupiah (`int64`/`BIGINT`).
- Wallet balance hanya berubah melalui create/edit/void/restore transaction.
- Update balance menggunakan atomic SQL expression di dalam DB transaction.
- Negative balance dipertahankan untuk MVP.
- Timezone business default `Asia/Jakarta` melalui `APP_TIMEZONE`.
- Staff tidak dapat mengirim tanggal; server memakai tanggal hari berjalan.
- Owner dapat mengirim `date` untuk backdate.

## Migration production

Migration di bawah adalah upgrade dari schema lama yang masih memakai
`wallets.user_id`/`categories.user_id`. Jalankan pada backup/clone terlebih
dahulu, bukan langsung pada database production.

1. Hentikan API lama atau aktifkan maintenance mode agar tidak ada write saat
   backfill berjalan.
2. Buat backup:

```bash
mysqldump --single-transaction --routines --triggers \
  -h "$DB_HOST" -P "${DB_PORT:-3306}" -u "$DB_USER" -p \
  "$DB_NAME" > cashmate-before-tenant-migration-$(date +%Y%m%d-%H%M%S).sql
```

3. Preflight data lama:

```sql
SELECT role, COUNT(*) AS total
FROM users
WHERE deleted_at IS NULL
GROUP BY role;

SELECT id, balance
FROM wallets
WHERE balance <> TRUNCATE(balance, 0);

SELECT id, amount
FROM transactions
WHERE amount <> TRUNCATE(amount, 0);
```

Hentikan proses jika ada lebih dari satu Owner aktif yang perlu dipetakan atau
ada nilai pecahan rupiah. Migration `003` sengaja tidak melakukan pembulatan.

4. Jalankan script berurutan dari root repository. `mysql` akan meminta
   password secara interaktif:

```bash
mysql --protocol=tcp -h "$DB_HOST" -P "${DB_PORT:-3306}" \
  -u "$DB_USER" -p "$DB_NAME" < migrations/001_create_business_tenancy.sql

mysql --protocol=tcp -h "$DB_HOST" -P "${DB_PORT:-3306}" \
  -u "$DB_USER" -p "$DB_NAME" < migrations/002_backfill_legacy_business.sql

# Review hasil backfill dan rekonsiliasi saldo sebelum lanjut.
mysql --protocol=tcp -h "$DB_HOST" -P "${DB_PORT:-3306}" \
  -u "$DB_USER" -p "$DB_NAME" < migrations/003_convert_money_and_constraints.sql
```

Script `002` memetakan data lama ke satu Business `CashMate Legacy`, membuat
kategori rekonsiliasi, dan menambahkan transaction adjustment jika cached
balance berbeda dari ledger aktif.

5. Deploy binary/container API baru dengan `AUTO_MIGRATE=false`, lalu lakukan
   smoke test registration, login, staff, transaction, dashboard, dan tenant
   isolation.
6. Setelah smoke test berhasil, hapus kolom ownership lama:

```bash
mysql --protocol=tcp -h "$DB_HOST" -P "${DB_PORT:-3306}" \
  -u "$DB_USER" -p "$DB_NAME" < migrations/004_remove_legacy_ownership.sql
```

Migration ini tidak memiliki rollback otomatis. Jika ada kegagalan setelah DDL
dimulai, hentikan deployment dan restore backup/clone yang dibuat sebelum
migration. Jangan menjalankan ulang `001` atau `003` secara membabi buta karena
script versioned tersebut mengharapkan urutan satu kali.

Untuk database development kosong, gunakan `docker-compose.dev.yml`; file itu
memakai `AUTO_MIGRATE=true` dan tidak boleh diarahkan ke database production:

```bash
docker compose -f docker-compose.dev.yml up -d --build
```

## Test

Unit/route tests tanpa database:

```bash
GOCACHE=/tmp/cashmate-go-build go test ./...
GOCACHE=/tmp/cashmate-go-build go vet ./...
```

Integration test membutuhkan database dengan nama yang mengandung `_test`:

```bash
CASHMATE_TEST_DSN='cashmate:cashmate@tcp(127.0.0.1:3307)/cashmate_test?charset=utf8mb4&parseTime=True&loc=Asia%2FJakarta' \
GOCACHE=/tmp/cashmate-go-build go test ./integration -v
```

Integration flow menguji dua Business, IDOR, Staff RBAC, Staff daily history,
Owner visibility, edit/void, dan balance consistency.
