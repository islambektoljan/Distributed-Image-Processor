#!/bin/bash

# Keycloak Setup Script for ImageProcessor
# Этот скрипт запускается внутри отдельного контейнера 'setup-keycloak'
# и обращается к Keycloak по внутреннему hostname 'keycloak'.

ADMIN_USER="admin"
ADMIN_PASSWORD="admin"
KEYCLOAK_URL="http://keycloak:8080" # <-- ИСПОЛЬЗУЕМ ВНУТРЕННЕЕ ИМЯ

echo "Starting Keycloak configuration using internal Keycloak URL: $KEYCLOAK_URL"

# Wait for Keycloak to be fully ready (health check уже сработал, но для kcadm нужно больше)
echo "Waiting for Keycloak service to stabilize..."
sleep 5 # Краткое ожидание после Healthcheck

# Login to Keycloak Admin CLI
echo "Logging in to Keycloak Admin CLI..."
/usr/bin/kcadm.sh config credentials \
  --server $KEYCLOAK_URL \
  --realm master \
  --user $ADMIN_USER \
  --password $ADMIN_PASSWORD

# Create ImageProcessor Realm
echo "Creating ImageProcessor realm..."
/usr/bin/kcadm.sh create realms \
  -s realm=ImageProcessor \
  -s enabled=true

# Create api-gateway-client
echo "Creating api-gateway-client..."
/usr/bin/kcadm.sh create clients \
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
/usr/bin/kcadm.sh create users \
  -r ImageProcessor \
  -s username=user \
  -s enabled=true

# Set password for test user
echo "Setting password for user..."
/usr/bin/kcadm.sh set-password \
  -r ImageProcessor \
  --username user \
  --new-password password

echo "Keycloak configuration completed successfully!"