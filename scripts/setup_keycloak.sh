#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status

# Keycloak Setup Script for ImageProcessor
# Этот скрипт запускается внутри отдельного контейнера 'setup-keycloak'
# и обращается к Keycloak по внутреннему hostname 'keycloak'.

ADMIN_USER="admin"
ADMIN_PASSWORD="admin"
KEYCLOAK_URL="http://keycloak:8080"
KCADM="/opt/keycloak/bin/kcadm.sh"

echo "Starting Keycloak configuration using internal Keycloak URL: $KEYCLOAK_URL"

# ------------------------------
# Robust Wait Loop using kcadm.sh
# ------------------------------
echo "Waiting for Keycloak service to stabilize (polling with kcadm)..."
MAX_RETRIES=40 # 40 attempts x 2 seconds = 80 seconds timeout
RETRY_COUNT=0

# Loop until kcadm.sh successfully connects and sets credentials to the master realm
until $KCADM config credentials --server $KEYCLOAK_URL --realm master --user $ADMIN_USER --password $ADMIN_PASSWORD > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Keycloak service not ready after $MAX_RETRIES attempts."
        exit 1
    fi
    # Wait 2 seconds between checks
    echo "Keycloak not ready yet. Waiting 2 seconds... (Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "Keycloak is now authenticated and ready. Proceeding with realm configuration..."

# Create ImageProcessor Realm
echo "Creating ImageProcessor realm..."
$KCADM create realms \
  -s realm=ImageProcessor \
  -s enabled=true

# Create api-gateway-client
echo "Creating api-gateway-client..."
$KCADM create clients \
  -r ImageProcessor \
  -s clientId=api-gateway-client \
  -s enabled=true \
  -s publicClient=true \
  -s directAccessGrantsEnabled=true \
  -s standardFlowEnabled=true \
  -s 'redirectUris=["http://localhost:3000/*"]' \
  -s 'webOrigins=["*"]'

# Create test user
echo "Creating test user 'user'..."
$KCADM create users \
  -r ImageProcessor \
  -s username=user \
  -s enabled=true

# Set password for test user
echo "Setting password for user..."
$KCADM set-password \
  -r ImageProcessor \
  --username user \
  --new-password password

echo "Keycloak configuration completed successfully!"