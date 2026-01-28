# Pushem

A self-hosted notification service designed as an alternative to ntfy.sh. Send notifications via a simple REST API and receive them as Web Push notifications on your devices (Android, iOS PWA, Desktop).

## Features

- **Simple REST API**: Send notifications with a single HTTP POST request
- **Topic-based subscriptions**: Subscribe to specific topics to receive targeted notifications
- **Cross-platform**: Works on Android, iOS (PWA), and Desktop browsers
- **Zero-config database**: Uses SQLite for easy deployment
- **Web Push Protocol**: Standards-compliant push notifications with VAPID authentication
- **No WebSockets required**: Uses HTTP/2 Web Push exclusively

## Architecture

```
User/Script → POST /publish/{topic} → Go Backend → Push Service → User Device
                                           ↓
                                      SQLite DB
```

## Quick Start with Docker/Podman (Recommended)

The easiest way to run Pushem is with Docker or Podman:

```bash
# Clone the repository
git clone https://github.com/yourusername/pushem.git
cd pushem

# Create data directory for persistence
mkdir data

# Build and run with Docker Compose
docker-compose up -d

# OR with Podman Compose
podman-compose up -d

# OR with Podman directly
podman build -t pushem .
podman run -d --name pushem -p 8080:8080 \
  -v ./data:/app/data:Z \
  -e PORT=8080 \
  -e STATIC_DIR=../web/dist \
  pushem sh -c "cd data && ../pushem"
```

Access the web interface at `http://localhost:8080`

The database and VAPID keys will be persisted in the `./data` directory.

### Production Setup with Caddy (Automatic HTTPS)

For production deployments, use Caddy for automatic HTTPS with Let's Encrypt:

1. **Edit the Caddyfile** and replace `pushem.example.com` with your domain:
   ```bash
   nano Caddyfile
   # Change pushem.example.com to your actual domain
   # Change admin@example.com to your email
   ```

2. **Make sure your domain points to your server** (DNS A record)

3. **Run with Caddy enabled**:

   ```bash
   # Docker Compose with profiles (if supported)
   docker-compose --profile caddy up -d

   # Podman Compose (use standalone file)
   podman-compose -f docker-compose.caddy.yml up -d

   # Docker Compose (alternative method)
   docker-compose -f docker-compose.caddy.yml up -d
   ```

4. **Access your site**:
   - Your site will be available at `https://your-domain.com`
   - Caddy automatically obtains and renews SSL certificates
   - HTTP automatically redirects to HTTPS
   - HTTP/3 support is enabled

**Caddy Features:**
- ✓ Automatic HTTPS with Let's Encrypt
- ✓ Automatic certificate renewal
- ✓ HTTP/2 and HTTP/3 support
- ✓ Security headers (HSTS, CSP, etc.)
- ✓ Gzip compression
- ✓ Access logging

**View Caddy logs:**
```bash
# With Docker
docker-compose -f docker-compose.caddy.yml logs -f caddy

# With Podman
podman-compose -f docker-compose.caddy.yml logs -f caddy
```

**Configuration Files:**
- `docker-compose.yml` - Default setup (HTTP only, direct access)
- `docker-compose.simple.yml` - Simple Podman-compatible version
- `docker-compose.caddy.yml` - Production setup with Caddy HTTPS
- `Caddyfile` - Caddy reverse proxy configuration

### Container Management

```bash
# View logs (Docker)
docker-compose logs -f

# View logs (Podman)
podman logs -f pushem

# Stop the service (Docker)
docker-compose down

# Stop the service (Podman)
podman stop pushem && podman rm pushem

# Rebuild after code changes (Docker)
docker-compose up -d --build

# Rebuild after code changes (Podman)
podman build -t pushem . && podman restart pushem
```

## Manual Installation (Without Docker)

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- GCC (for SQLite driver)
- A modern web browser with Service Worker support
- HTTPS for production (required for Service Workers; localhost works in development)

### 1. Clone the repository

```bash
git clone https://github.com/yourusername/pushem.git
cd pushem
```

### 2. Build the frontend

```bash
cd web
npm install
npm run build
cd ..
```

### 3. Build the backend

```bash
go build -o pushem cmd/server/main.go
```

### 4. Run the server

```bash
./pushem
```

The server will:
- Start on port 8080 (or PORT environment variable)
- Generate VAPID keys on first run and save them to `vapid_keys.json`
- Create a SQLite database file `pushem.db`
- Serve the web interface at `http://localhost:8080`

## Usage

### Web Interface

1. Open `http://localhost:8080` in your browser
2. Enter a topic name (e.g., "my-alerts")
3. Click "Subscribe" and grant notification permissions
4. You're subscribed! Leave the tab open or install as a PWA

### iOS Setup

On iOS, Web Push requires the app to be installed as a PWA:

1. Open Safari and navigate to your Pushem instance
2. Tap the Share button
3. Select "Add to Home Screen"
4. Open Pushem from your home screen
5. Subscribe to topics as normal

### Publishing Notifications

#### Simple text notification

```bash
curl -X POST http://localhost:8080/publish/my-alerts \
  -d "Hello from Pushem!"
```

#### JSON notification with title and custom fields

```bash
curl -X POST http://localhost:8080/publish/my-alerts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Alert",
    "message": "Something important happened!",
    "click_url": "https://example.com"
  }'
```

### API Endpoints

#### `GET /vapid-public-key`

Returns the VAPID public key for client-side subscription.

**Response:**
```json
{
  "publicKey": "BFxw..."
}
```

#### `POST /subscribe/{topic}`

Subscribe a device to receive notifications for a topic.

**Body:**
```json
{
  "endpoint": "https://fcm.googleapis.com/fcm/send/...",
  "keys": {
    "p256dh": "...",
    "auth": "..."
  }
}
```

**Response:**
```json
{
  "status": "subscribed"
}
```

#### `POST /publish/{topic}`

Send a notification to all subscribers of a topic.

**Body (Plain Text):**
```
Hello World!
```

**Body (JSON):**
```json
{
  "title": "Notification Title",
  "message": "Notification body text",
  "click_url": "https://example.com"
}
```

**Response:**
```json
{
  "status": "published",
  "sent": 5,
  "failed": 0
}
```

## Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)
- `STATIC_DIR`: Path to frontend static files (default: web/dist)
- `MESSAGE_RETENTION_DAYS`: Number of days to keep message history (default: 7)
- `CLEANUP_INTERVAL_HOURS`: Hours between automatic cleanup runs (default: 24)

**Message History Cleanup:**

Pushem automatically cleans up old messages to prevent database bloat. By default:
- Messages older than 7 days are automatically deleted
- Cleanup runs every 24 hours
- First cleanup runs 1 minute after server start

To customize:
```bash
# Keep messages for 30 days, cleanup every 6 hours
docker run -e MESSAGE_RETENTION_DAYS=30 -e CLEANUP_INTERVAL_HOURS=6 ...
```

To disable automatic cleanup, set retention to a very high value:
```bash
docker run -e MESSAGE_RETENTION_DAYS=36500 ...  # ~100 years
```

### Files

- `pushem.db`: SQLite database containing subscriptions
- `vapid_keys.json`: VAPID key pair for Web Push authentication

## Development

### Run frontend in development mode

```bash
cd web
npm run dev
```

This starts Vite dev server on `http://localhost:5173` with hot reload.

### Run backend

```bash
go run cmd/server/main.go
```

## Security Considerations

Pushem includes comprehensive security features:

- **Input Validation**: All API endpoints validate and sanitize user input
  - Topic name validation (alphanumeric, length limits, reserved names)
  - Message content validation (UTF-8, length limits, null byte detection)
  - SSRF protection for subscription endpoints (blocks private IPs)
  - Secret key strength validation

- **Topic Protection**: Optional secret keys to protect topics from unauthorized access
  ```bash
  # Protect a topic
  curl -X POST http://localhost:8080/protect/my-topic \
    -d '{"secret": "your-secret-key"}'

  # Publish with authentication
  curl -X POST http://localhost:8080/publish/my-topic \
    -H "X-Pushem-Key: your-secret-key" \
    -d "Protected message"
  ```

- **HTTPS Required**: For production deployments, HTTPS is mandatory for Service Workers
  - Use Caddy for automatic Let's Encrypt certificates (see CADDY_SETUP.md)

- **VAPID Keys**: Keep `vapid_keys.json` secure. Losing these keys will invalidate all subscriptions
  ```bash
  chmod 600 data/vapid_keys.json
  ```

- **Rate Limiting**: Use Caddy or a reverse proxy for rate limiting in production

For detailed security information, see [SECURITY.md](SECURITY.md)

## Project Structure

```
Pushem/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── api/
│   │   └── handlers.go      # HTTP handlers
│   ├── db/
│   │   └── sqlite.go        # Database layer
│   └── webpush/
│       └── service.go       # Web Push service
├── web/                     # Frontend React app
│   ├── src/
│   ├── public/
│   └── dist/                # Built frontend files
├── go.mod
├── go.sum
└── README.md
```

## Use Cases

- **Server monitoring**: Get alerts when services go down
- **CI/CD notifications**: Notify when builds complete
- **Home automation**: Receive alerts from IoT devices
- **Personal notifications**: Send yourself reminders or updates
- **Team notifications**: Share alerts with team members subscribed to the same topic

## Examples

### Shell script notification

```bash
#!/bin/bash
# Send notification when a long process completes
make build && \
  curl -X POST http://localhost:8080/publish/builds \
    -H "Content-Type: application/json" \
    -d '{"title":"Build Complete","message":"Your project built successfully!"}'
```

### Python script

```python
import requests

def send_notification(topic, title, message, click_url=None):
    payload = {
        "title": title,
        "message": message
    }
    if click_url:
        payload["click_url"] = click_url

    response = requests.post(
        f"http://localhost:8080/publish/{topic}",
        json=payload
    )
    return response.json()

# Usage
send_notification("alerts", "Test", "Hello from Python!")
```

## Troubleshooting

### Notifications not appearing

1. Check that notification permissions are granted in browser settings
2. Verify the Service Worker is registered (check browser DevTools → Application → Service Workers)
3. Ensure you're using HTTPS in production (or localhost in development)
4. Check browser console for errors

### iOS not working

1. Make sure you've installed the PWA (Add to Home Screen)
2. Open the app from the home screen icon, not Safari
3. iOS requires the app to be running for notifications to appear

### Database errors

Delete `pushem.db` and restart the server to recreate the database.

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

Built with:
- [Go](https://golang.org/)
- [Chi Router](https://github.com/go-chi/chi)
- [webpush-go](https://github.com/SherClockHolmes/webpush-go)
- [React](https://react.dev/)
- [Vite](https://vite.dev/)
- [Tailwind CSS](https://tailwindcss.com/)
# pushem
