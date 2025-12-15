#!/bin/bash

# Keycloak Setup Script for ImageProcessor
# This script configures Keycloak with realm, client, and test user

KEYCLOAK_CONTAINER="distributedimageprocessor-keycloak-1"
ADMIN_USER="admin"
ADMIN_PASSWORD="admin"
KEYCLOAK_URL="http://localhost:8080"

echo "Starting Keycloak configuration..."

# Wait for Keycloak to be ready
echo "Waiting for Keycloak to start..."
sleep 10

# Login to Keycloak Admin CLI
echo "Logging in to Keycloak Admin CLI..."
docker exec $KEYCLOAK_CONTAINER /opt/keycloak/bin/kcadm.sh config credentials \
  --server $KEYCLOAK_URL \
  --realm master \
  --user $ADMIN_USER \
  --password $ADMIN_PASSWORD

# Create ImageProcessor Realm
echo "Creating ImageProcessor realm..."
docker exec $KEYCLOAK_CONTAINER /opt/keycloak/bin/kcadm.sh create realms \
  -s realm=ImageProcessor \
  -s enabled=true

# Create api-gateway-client
echo "Creating api-gateway-client..."
docker exec $KEYCLOAK_CONTAINER /opt/keycloak/bin/kcadm.sh create clients \
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
docker exec $KEYCLOAK_CONTAINER /opt/keycloak/bin/kcadm.sh create users \
  -r ImageProcessor \
  -s username=user \
  -s enabled=true

# Set password for test user
echo "Setting password for user..."
docker exec $KEYCLOAK_CONTAINER /opt/keycloak/bin/kcadm.sh set-password \
  -r ImageProcessor \
  --username user \
  --new-password password

echo "Keycloak configuration completed successfully!"
echo ""
echo "Realm: ImageProcessor"
echo "Client ID: api-gateway-client"
echo "Test User: user / password"
echo ""
echo "JWKS URL: http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/certs"
echo "Token URL: http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/token"
echo ""
echo "To get a token, run:"
echo "curl -X POST 'http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/token' \\"
echo "  -H 'Content-Type: application/x-www-form-urlencoded' \\"
echo "  -d 'grant_type=password' \\"
echo "  -d 'client_id=api-gateway-client' \\"
echo "  -d 'username=user' \\"
echo "  -d 'password=password'"
