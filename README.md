# Distributed Image Processor

A high-performance distributed image processing system built with Go, featuring async workers, object storage, and OAuth2 authentication.

## Architecture

- **API Gateway**: REST API for image upload and retrieval
- **Worker Service**: Background image processing (resize, grayscale)
- **PostgreSQL**: Metadata storage
- **Redis**: Caching layer
- **MinIO**: Object storage for images
- **RabbitMQ**: Message queue for async processing
- **Keycloak**: OAuth2/OIDC authentication

## Development Setup

### Prerequisites
- Docker & Docker Compose
- Go 1.23+ (for local development)

### Quick Start (Development Mode)

1. **Start infrastructure:**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

2. **Setup Keycloak:**
   ```bash
   bash scripts/setup_keycloak.sh
   ```

3. **Run API Gateway locally:**
   ```bash
   go run cmd/api-gateway/main.go
   ```

4. **Run Worker locally:**
   ```bash
   go run cmd/worker/main.go
   ```

5. **Get access token:**
   ```bash
   curl -X POST 'http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/token' \
     -H 'Content-Type: application/x-www-form-urlencoded' \
     -d 'grant_type=password' \
     -d 'client_id=api-gateway-client' \
     -d 'username=user' \
     -d 'password=password'
   ```

6. **Upload image:**
   ```bash
   curl -X POST http://localhost:3000/api/v1/upload \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -F "image=@test.png"
   ```

## Production Deployment

### Build and Run

1. **Build Docker images and start all services:**
   ```bash
   docker-compose -f docker-compose.prod.yml up -d --build
   ```

2. **Setup Keycloak (first time only):**
   ```bash
   # Wait for Keycloak to start (30 seconds)
   sleep 30
   bash scripts/setup_keycloak.sh
   ```

3. **View logs:**
   ```bash
   docker-compose -f docker-compose.prod.yml logs -f api-gateway worker
   ```

4. **Scale workers:**
   ```bash
   docker-compose -f docker-compose.prod.yml up -d --scale worker=5
   ```

### Production Configuration

All services communicate through internal Docker network. Environment variables:

- `POSTGRES_URL`: Database connection string
- `REDIS_URL`: Redis server address
- `MINIO_ENDPOINT`: MinIO server address
- `RABBITMQ_URL`: RabbitMQ connection string
- `KEYCLOAK_URL`: Keycloak server URL

## API Endpoints

### Health Check
```bash
GET /health
```

### Upload Image (Protected)
```bash
POST /api/v1/upload
Authorization: Bearer {token}
Content-Type: multipart/form-data

Form Data:
- image: file (JPEG/PNG, max 10MB)
```

### Get Image Status (Protected)
```bash
GET /api/v1/images/:id
Authorization: Bearer {token}
```

## Testing

1. **Get token:**
   ```bash
   TOKEN=$(curl -s -X POST 'http://localhost:8080/realms/ImageProcessor/protocol/openid-connect/token' \
     -H 'Content-Type: application/x-www-form-urlencoded' \
     -d 'grant_type=password' \
     -d 'client_id=api-gateway-client' \
     -d 'username=user' \
     -d 'password=password' | jq -r '.access_token')
   ```

2. **Upload test image:**
   ```bash
   curl -X POST http://localhost:3000/api/v1/upload \
     -H "Authorization: Bearer $TOKEN" \
     -F "image=@test.png"
   ```

3. **Check status:**
   ```bash
   curl http://localhost:3000/api/v1/images/{IMAGE_ID} \
     -H "Authorization: Bearer $TOKEN"
   ```

## Image Processing

Workers automatically:
1. Download images from `raw-images` bucket
2. Resize to 800px width (maintaining aspect ratio)
3. Apply grayscale filter
4. Save to `processed-images` bucket
5. Update database status to "completed"
6. Invalidate Redis cache

## Management Interfaces

- **MinIO Console**: http://localhost:9001 (minioadmin/minioadmin)
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Keycloak Admin**: http://localhost:8080/admin (admin/admin)

## Troubleshooting

### Services won't start
```bash
docker-compose -f docker-compose.prod.yml down -v
docker-compose -f docker-compose.prod.yml up -d --build
```

### Check service logs
```bash
docker-compose -f docker-compose.prod.yml logs [service-name]
```

### Reset database
```bash
docker-compose -f docker-compose.prod.yml down -v
docker-compose -f docker-compose.prod.yml up -d
```

## Project Structure

```
.
├── cmd/
│   ├── api-gateway/    # API Gateway entrypoint
│   ├── worker/         # Worker Service entrypoint
│   └── migrate/        # Database migrations
├── internal/
│   ├── config/         # Configuration management
│   ├── handler/        # HTTP handlers
│   ├── models/         # Data models
│   ├── queue/          # RabbitMQ client
│   ├── storage/        # MinIO client
│   └── worker/         # Image processing logic
├── pkg/
│   ├── database/       # Database clients (Postgres, Redis)
│   └── security/       # Keycloak JWT middleware
├── docs/               # Documentation
├── scripts/            # Setup scripts
├── Dockerfile.api      # API Gateway image
├── Dockerfile.worker   # Worker Service image
└── docker-compose.*.yml
```

## License

MIT