# Pushem - Docker/Podman Build Complete ✅

## Summary

Successfully implemented the complete Pushem notification service with Docker/Podman support. The application is now containerized and ready to deploy!

## What Was Fixed for Docker/Podman

1. **Multi-stage Dockerfile** created with:
   - Node.js 20 for frontend build (Vite 7 requirement)
   - Go 1.23 for backend build with CGO support for SQLite
   - Minimal Alpine Linux runtime image

2. **Fixed Go version** in go.mod (was 1.25.5, now 1.23)

3. **Added STATIC_DIR environment variable** to support flexible paths

4. **Created docker-compose.yml** with proper volume mounting for data persistence

5. **Tested with Podman** - builds and runs successfully!

## Quick Start

### Using Podman (Recommended for your system)

```bash
# Build the image
podman build -t pushem .

# Create data directory
mkdir -p data

# Run the container
podman run -d --name pushem -p 8080:8080 \
  -v ./data:/app/data:Z \
  -e PORT=8080 \
  -e STATIC_DIR=../web/dist \
  pushem sh -c "cd data && ../pushem"

# View logs
podman logs -f pushem

# Test the API
curl http://localhost:8080/vapid-public-key
```

### Using Docker Compose

```bash
mkdir -p data
docker-compose up -d
docker-compose logs -f
```

### Using Podman Compose

```bash
mkdir -p data
podman-compose up -d
podman-compose logs -f
```

## Verification

The container successfully:
- ✅ Builds both frontend (React PWA) and backend (Go)
- ✅ Generates VAPID keys on first run
- ✅ Persists database and keys to `./data` volume
- ✅ Serves web interface on port 8080
- ✅ Responds to API requests
- ✅ Frontend loads correctly

## Testing

Use the included test script:

```bash
./test-notification.sh my-alerts
```

Or test manually:

```bash
# 1. Open http://localhost:8080 and subscribe to a topic

# 2. Send a notification
curl -X POST http://localhost:8080/publish/my-alerts \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","message":"It works!"}'
```

## Project Structure

```
Pushem/
├── Dockerfile              # Multi-stage build
├── docker-compose.yml      # Docker Compose config
├── .dockerignore          # Docker ignore rules
├── cmd/server/main.go     # Go backend
├── internal/              # Go packages
├── web/                   # React frontend
│   ├── src/App.tsx       # Main UI
│   ├── public/sw.js      # Service Worker
│   └── dist/             # Built frontend
├── data/                  # Persisted data (gitignored)
│   ├── pushem.db         # SQLite database
│   └── vapid_keys.json   # VAPID keypair
└── test-notification.sh   # Test script
```

## Container Image Details

- **Base Images**: Node 20 Alpine, Go 1.23 Alpine, Alpine Latest
- **Final Image Size**: ~20-30 MB (minimal Alpine + binary)
- **Exposed Port**: 8080
- **Volumes**: `/app/data` for persistence
- **Environment Variables**:
  - `PORT`: Server port (default: 8080)
  - `STATIC_DIR`: Frontend files path (default: ../web/dist)

## Development Workflow

### Rebuild after code changes

```bash
# Podman
podman build -t pushem .
podman stop pushem && podman rm pushem
podman run -d --name pushem -p 8080:8080 \
  -v ./data:/app/data:Z \
  -e PORT=8080 \
  -e STATIC_DIR=../web/dist \
  pushem sh -c "cd data && ../pushem"

# Docker Compose
docker-compose up -d --build
```

### Frontend development with hot reload

```bash
# Terminal 1: Run backend
podman run -d --name pushem-backend -p 8080:8080 \
  -v ./data:/app/data:Z \
  pushem sh -c "cd data && ../pushem"

# Terminal 2: Run frontend dev server
cd web
npm run dev
# Frontend at http://localhost:5173 (proxies API to :8080)
```

## Production Deployment

### HTTPS Requirement

For production, you MUST use HTTPS (Service Workers requirement):

```bash
# Use a reverse proxy like nginx or traefik
# Example with nginx:

server {
    listen 443 ssl http2;
    server_name pushem.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Systemd Service (with Podman)

Create `/etc/systemd/system/pushem.service`:

```ini
[Unit]
Description=Pushem Notification Service
After=network-online.target

[Service]
Type=simple
User=pushem
WorkingDirectory=/opt/pushem
ExecStartPre=/usr/bin/podman rm -f pushem
ExecStart=/usr/bin/podman run --name pushem -p 8080:8080 \
  -v /opt/pushem/data:/app/data:Z \
  -e PORT=8080 \
  -e STATIC_DIR=../web/dist \
  pushem sh -c "cd data && ../pushem"
ExecStop=/usr/bin/podman stop pushem
Restart=always

[Install]
WantedBy=multi-user.target
```

## Next Steps

1. **Deploy to your server** using the Podman commands above
2. **Set up HTTPS** with a reverse proxy
3. **Test notifications** from your applications
4. **Optional**: Add authentication for the publish endpoint
5. **Optional**: Set up monitoring and alerts

## Files Created/Modified

- ✅ Dockerfile (multi-stage build)
- ✅ docker-compose.yml (orchestration)
- ✅ .dockerignore (build optimization)
- ✅ cmd/server/main.go (added STATIC_DIR env var)
- ✅ go.mod (fixed version to 1.23)
- ✅ test-notification.sh (testing script)
- ✅ README.md (updated with Docker/Podman instructions)

## Success!

Your Pushem service is now fully containerized and tested. The Docker/Podman build eliminates all host dependencies (like GCC for SQLite) and makes deployment consistent across any environment.

Ready to deploy! 🚀
