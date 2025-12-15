# Keycloak Setup and Testing Guide

## Step 1: Run Keycloak Setup Script

The setup script will configure Keycloak with the necessary realm, client, and test user.

### For Unix/Linux/Mac:
```bash
cd scripts
bash setup_keycloak.sh
```

### For Windows (PowerShell):
You can run the commands manually in Git Bash or WSL:
```powershell
bash scripts/setup_keycloak.sh
```

Alternatively, run each command individually:

```powershell
docker exec distributedimageprocessor-keycloak-1 /opt/keycloak/bin/kcadm.sh config credentials `
  --server http://localhost:8080 `
  --realm master `
  --user admin `
  --password admin

docker exec distributedimageprocessor-keycloak-1 /opt/keycloak/bin/kcadm.sh create realms `
  -s realm=ImageProcessor `
  -s enabled=true

docker exec distributedimageprocessor-keycloak-1 /opt/keycloak/bin/kcadm.sh create clients `
  -r ImageProcessor `
  -s clientId=api-gateway-client `
  -s enabled=true `
  -s publicClient=true `
  -s directAccessGrantsEnabled=true `
  -s standardFlowEnabled=true `
  -s 'redirectUris=["http://localhost:3000/*"]' `
  -s 'webOrigins=["*"]'

docker exec distributedimageprocessor-keycloak-1 /opt/keycloak/bin/kcadm.sh create users `
  -r ImageProcessor `
  -s username=user `
  -s enabled=true

docker exec distributedimageprocessor-keycloak-1 /opt/keycloak/bin/kcadm.sh set-password `
  -r ImageProcessor `
  --username user `
  --new-password password
```

## Step 2: Verify Keycloak Configuration

Visit http://localhost:8080/admin and login with `admin` / `admin`.

Verify that:
- Realm `ImageProcessor` exists
- Client `api-gateway-client` is configured
- User `user` exists

## Step 3: Obtain Authentication Token

Use the OAuth 2.0 Password Grant flow to get a JWT token:

```bash
curl -X POST 'http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=password' \
  -d 'client_id=api-gateway-client' \
  -d 'username=user' \
  -d 'password=password'
```

The response will contain an `access_token`:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 300,
  "refresh_expires_in": 1800,
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer"
}
```

## Step 4: Test Protected Endpoints

### Test Health Endpoint (No auth required)
```bash
curl http://localhost:3000/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "api-gateway"
}
```

### Test Upload Endpoint (Auth required)
```bash
export TOKEN="<your_access_token_from_step_3>"

curl -X POST http://localhost:3000/api/v1/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@test_image.png"
```

### Test Get Image Endpoint (Auth required)
```bash
curl http://localhost:3000/api/v1/images/<image_id> \
  -H "Authorization: Bearer $TOKEN"
```

### Test Without Token (Should return 401)
```bash
curl http://localhost:3000/api/v1/upload \
  -F "image=@test_image.png"
```

Expected response:
```json
{
  "error": "Authorization header required"
}
```

## Keycloak Configuration Details

- **Realm:** ImageProcessor
- **Client ID:** api-gateway-client
- **Client Type:** Public (no client secret required)
- **Allowed Flows:** Direct Access Grants (Password Grant), Standard Flow
- **Test User:** user / password

## JWKS URL

The API Gateway validates tokens using the public keys from:
```
http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/certs
```

## Token Information

Tokens are valid for 5 minutes by default. After expiration, you'll need to request a new token or use the refresh token.
