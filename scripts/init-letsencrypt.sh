#!/usr/bin/env bash
# ============================================================
# Setup SSL HTTPS dengan Let's Encrypt (webroot + nginx)
#
# Cara pakai:
#   ./scripts/init-letsencrypt.sh  atau
#   ./scripts/init-letsencrypt.sh example.com you@email.com
#
# Prasyarat:
#   - Domain sudah mengarah (DNS A record) ke IP VPS ini.
#   - Port 80 sudah terbuka di firewall.
# ============================================================
set -euo pipefail

# Baca .env kalau ada
if [ -f .env ]; then
  while IFS='=' read -r key value; do
    case "$key" in
      ''|\#*) continue ;;
      *) export "$key=$value" ;;
    esac
  done < .env
fi

DOMAIN="${1:-${APP_DOMAIN:-}}"
EMAIL="${2:-admin@${DOMAIN}}"

if [ -z "$DOMAIN" ]; then
  echo "ERROR: Domain tidak ditemukan."
  echo "  Jalankan:  $0 example.com [you@email.com]"
  echo "  atau isi  APP_DOMAIN=  di file .env"
  exit 1
fi

echo "==> Menjalankan container api + nginx..."
# Production uses the teacher-provided MySQL database; there is intentionally
# no production `db` service in docker-compose.yml.
docker compose up -d api nginx

echo "==> Mendapatkan sertifikat Let's Encrypt untuk $DOMAIN ..."
docker compose run --rm --entrypoint certbot certbot certonly \
  --webroot \
  -w /var/www/certbot \
  -d "$DOMAIN" \
  --email "$EMAIL" \
  --agree-tos \
  --no-eff-email \
  --force-renewal

echo "==> Mengaktifkan konfigurasi HTTPS..."
sed "s/__DOMAIN__/$DOMAIN/g" nginx/templates/cashmate-ssl.conf.tpl > nginx/conf.d/cashmate-ssl.conf
docker compose exec nginx nginx -s reload

echo ""
echo "Selesai! API sekarang bisa diakses di:"
echo "  https://$DOMAIN/api/health"
echo ""
echo "Renewal sertifikat berjalan otomatis tiap 12 jam via service certbot."
