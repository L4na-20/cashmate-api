#!/usr/bin/env bash
# ============================================================
# Script deploy yang dijalankan DI DALAM VPS oleh GitHub Actions.
#
# Yang dilakukan:
#   1. Git pull source code terbaru (branch main)
#   2. Memastikan file .env ada (tidak pernah di-commit)
#   3. docker compose -f docker-compose.yml up -d --build
#   4. Health check API
#
# Variabel yang dibutuhkan (dikirim oleh GitHub Actions):
#   REPO_URL     e.g. git@github.com:username/cashmate-api.git
#   DEPLOY_PATH  e.g. /home/ubuntu/cashmate-api
#   BRANCH       default: main
# ============================================================
set -euo pipefail

REPO_URL="${REPO_URL:?REPO_URL wajib diisi}"
DEPLOY_PATH="${DEPLOY_PATH:?DEPLOY_PATH wajib diisi}"
BRANCH="${BRANCH:-main}"

# Terima host key GitHub saat clone pertama kali (non-interaktif)
export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new"

echo "==> [1/4] Update source code (branch $BRANCH)..."
mkdir -p "$DEPLOY_PATH"
if [ ! -d "$DEPLOY_PATH/.git" ]; then
  echo "    Repo belum ada, clone..."
  git clone --branch "$BRANCH" "$REPO_URL" "$DEPLOY_PATH"
else
  git -C "$DEPLOY_PATH" fetch origin "$BRANCH"
  git -C "$DEPLOY_PATH" checkout "$BRANCH"
  git -C "$DEPLOY_PATH" reset --hard "origin/$BRANCH"
fi

cd "$DEPLOY_PATH"

echo "==> [2/4] Memeriksa file .env..."
if [ ! -f .env ]; then
  echo "ERROR: $DEPLOY_PATH/.env tidak ditemukan!"
  echo ""
  echo "Buat dulu di VPS:"
  echo "  cd $DEPLOY_PATH"
  echo "  cp .env.example .env"
  echo "  nano .env"
  echo ""
  echo "Lalu isi minimal: DB_USER, DB_PASSWORD, DB_NAME, JWT_SECRET,"
  echo "JWT_REFRESH_SECRET, CORS_ORIGINS, APP_DOMAIN, dst."
  exit 1
fi

echo "==> [3/4] Build image & jalankan container..."
# Production deliberately uses the database outside Docker. Keep this file
# explicit so a local COMPOSE_FILE or docker-compose.dev.yml cannot pull MySQL
# onto the VPS by accident.
docker compose -f docker-compose.yml up -d --build

echo "==> [4/4] Health check..."
for i in $(seq 1 30); do
  if curl -fsS http://localhost:8096/api/health >/dev/null 2>&1; then
    echo ""
    echo "✔ Deploy sukses! API sehat di http://localhost:8096/api/health"
    exit 0
  fi
  sleep 2
done

echo "ERROR: API tidak merespons setelah ~60 detik. Log terakhir:"
docker compose -f docker-compose.yml logs --tail=50 api
exit 1
