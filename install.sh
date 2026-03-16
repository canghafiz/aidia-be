#!/bin/bash
set -e

# Fix CRLF line endings (Windows)
sed -i 's/\r//' /home/deploy/ai-dia/.env

echo "======================================"
echo " AI-DIA INSTALLER"
echo "======================================"

source /home/deploy/ai-dia/.env

# ===========================================================
# Step 1: Setup Database
# ===========================================================
echo "[1/3] Setting up database..."
sudo -u postgres psql -c "ALTER USER ${DB_USER} PASSWORD '${DB_PASS}';" 2>/dev/null || true
sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME};" 2>/dev/null || true

# Allow Docker subnet ke PostgreSQL
DOCKER_SUBNET="${DB_HOST%.*}.0/16"
if ! sudo grep -q "$DOCKER_SUBNET" /etc/postgresql/*/main/pg_hba.conf 2>/dev/null; then
  echo "host    all    all    $DOCKER_SUBNET    md5" | sudo tee -a /etc/postgresql/*/main/pg_hba.conf
  sudo systemctl restart postgresql 2>/dev/null || true
  echo "✅ pg_hba.conf updated"
fi

sudo ufw allow from $DOCKER_SUBNET to any port 5432 2>/dev/null || true
sudo iptables -I INPUT 1 -s $DOCKER_SUBNET -p tcp --dport 5432 -j ACCEPT 2>/dev/null || true
sudo sed -i "s/listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/*/main/postgresql.conf
sudo systemctl restart postgresql 2>/dev/null || true
echo "✅ Database ${DB_NAME} ready"

# ===========================================================
# Step 2: Setup SSL + Reverse Proxy (aapanel)
# ===========================================================
echo "[2/3] Setting up SSL + Reverse Proxy..."

DOMAIN_API_HOST=$(echo $DOMAIN_API | sed 's|https\?://||' | sed 's|/.*||')

AAPANEL_NGINX="/www/server/nginx/sbin/nginx"
AAPANEL_VHOST="/www/server/panel/vhost/nginx"
AAPANEL_PROXY="/www/server/panel/vhost/nginx/proxy"
CERT_AAPANEL="/www/server/panel/vhost/cert"

if [ -f "$CERT_AAPANEL/$DOMAIN_API_HOST/fullchain.pem" ]; then
  echo "✅ SSL certificate found for $DOMAIN_API_HOST"

  # Write proxy config
  sudo mkdir -p $AAPANEL_PROXY/$DOMAIN_API_HOST
  sudo rm -f $AAPANEL_PROXY/$DOMAIN_API_HOST/proxy.conf
  sudo tee $AAPANEL_PROXY/$DOMAIN_API_HOST/proxy.conf > /dev/null <<EOF
location / {
    proxy_pass http://127.0.0.1:${APP_PORT};
    proxy_http_version 1.1;
    proxy_set_header Upgrade \$http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
}
EOF

  # Write vhost jika belum ada
  VHOST_FILE="$AAPANEL_VHOST/$DOMAIN_API_HOST.conf"
  if [ -f "$VHOST_FILE" ] && grep -q "proxy/$DOMAIN_API_HOST" "$VHOST_FILE" 2>/dev/null; then
    echo "ℹ️  Vhost $DOMAIN_API_HOST already configured, skipping..."
  else
    echo "⚙️  Writing vhost for $DOMAIN_API_HOST..."
    sudo tee $VHOST_FILE > /dev/null <<EOF
server {
    listen 80;
    server_name $DOMAIN_API_HOST;
    return 301 https://\$host\$request_uri;
}
server {
    listen 443 ssl http2;
    server_name $DOMAIN_API_HOST;
    ssl_certificate $CERT_AAPANEL/$DOMAIN_API_HOST/fullchain.pem;
    ssl_certificate_key $CERT_AAPANEL/$DOMAIN_API_HOST/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers EECDH+CHACHA20:EECDH+AES128:RSA+AES128:EECDH+AES256:RSA+AES256:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    add_header Strict-Transport-Security "max-age=31536000";
    include $AAPANEL_PROXY/$DOMAIN_API_HOST/*.conf;
    access_log /www/wwwlogs/$DOMAIN_API_HOST.log;
    error_log /www/wwwlogs/$DOMAIN_API_HOST.error.log;
}
EOF
  fi

  # Test & reload nginx
  if $AAPANEL_NGINX -t 2>/dev/null; then
    $AAPANEL_NGINX -s reload 2>/dev/null && echo "✅ Nginx reloaded"
  else
    echo "❌ Nginx config error:"
    $AAPANEL_NGINX -t 2>&1 || true
  fi

else
  echo ""
  echo "╔══════════════════════════════════════════════════════╗"
  echo "║   SSL certificate not found — Setup SSL first        ║"
  echo "╚══════════════════════════════════════════════════════╝"
  echo ""
  echo "  1. Login aapanel → Website → Add Site"
  echo "     Domain: $DOMAIN_API_HOST"
  echo ""
  echo "  2. SSL → Let's Encrypt → Apply"
  echo ""
  echo "  3. Run installer again:"
  echo "     cd /home/deploy/ai-dia && bash scripts/install.sh"
  echo ""
  exit 1
fi

# ===========================================================
# Step 3: Build & Run Docker
# ===========================================================
echo "[3/3] Building & starting Docker containers..."
cd /home/deploy/ai-dia

docker compose down 2>/dev/null || true
docker compose up -d --build

# Verify backend
echo "⏳ Verifying backend connection..."
sleep 10
BACKEND_LOG=$(docker logs ai-dia_backend --tail 5 2>&1)
if echo "$BACKEND_LOG" | grep -q "connection timed out\|connection refused\|no such host"; then
  echo "⚠️  Backend gagal konek ke database!"
  echo "    Cek log: docker logs ai-dia_backend"
else
  echo "✅ Backend running"
fi

# ===========================================================
# Done
# ===========================================================
echo ""
echo "======================================"
echo " ✅ INSTALL COMPLETE!"
echo " API: https://$DOMAIN_API_HOST"
echo "======================================"