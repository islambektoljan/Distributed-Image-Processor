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

# ------------------------------
# User Creation and Password Setup
# ------------------------------

# 1. Create test user and capture its ID from the output message.
echo "Creating test user 'user' and capturing ID..."

USER_CREATE_OUTPUT=$($KCADM create users -r ImageProcessor -s username=user -s enabled=true 2>&1)
USER_ID=$(echo "$USER_CREATE_OUTPUT" | grep "id '" | cut -d "'" -f 2)
  
if [ -z "$USER_ID" ]; then
    echo "Failed to create user or retrieve ID from output. Exiting."
    echo "Raw output of 'kcadm create users': $USER_CREATE_OUTPUT"
    exit 1
fi
echo "Successfully created and retrieved user ID: $USER_ID"

# 2. Set password for test user
echo "Setting password for user..."
$KCADM set-password \
  -r ImageProcessor \
  --username user \
  --new-password password 

# 3. Explicitly remove all required actions
echo "Manually forcing removal of all required actions for the test user to prevent invalid_grant..."
$KCADM update users/$USER_ID -r ImageProcessor -s 'requiredActions=[]'

echo "Forcing Keycloak to clear realm, user, and keys caches using kcadm create..."

$KCADM create clear-realm-cache -r ImageProcessor -s realm=ImageProcessor
$KCADM create clear-user-cache -r ImageProcessor -s realm=ImageProcessor
$KCADM create clear-keys-cache -r ImageProcessor -s realm=ImageProcessor

echo "Adding final sleep (5s) for stability..."
sleep 5

echo "Keycloak configuration completed successfully!"