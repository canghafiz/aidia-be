#!/bin/bash

sed -i 's/\r//' /home/deploy/ai-dia/.env
source /home/deploy/ai-dia/.env

echo "[1/2] Reset database..."
sudo -u postgres psql -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || echo "⚠️  Drop skipped or failed, continuing..."
sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME};" 2>/dev/null || echo "⚠️  Create skipped or failed, continuing..."
echo "✅ Database step done"

echo "[2/2] Build & run docker..."
cd /home/deploy/ai-dia

if ! docker compose down; then
  echo "❌ Failed to stop containers"
  exit 1
fi

if ! docker compose up -d --build; then
  echo "❌ Failed to build & start containers"
  exit 1
fi
echo "✅ Docker started"

echo "⏳ Waiting for migrate & backend to start..."
sleep 10

echo "📋 Backend logs:"
docker logs ai-dia_backend --tail 20

if docker logs ai-dia_backend --tail 20 2>&1 | grep -q "error\|failed\|refused"; then
  echo "⚠️  Backend mungkin ada masalah, cek log di atas"
else
  echo "✅ Done!"
fi