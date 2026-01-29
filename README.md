# Pushem

A self-hosted notification service designed as an alternative to ntfy.sh. Send notifications via a simple REST API and receive them as Web Push notifications on your devices (Android, iOS PWA, Desktop).

## Features

- **Simple REST API**: Send notifications with a single HTTP POST request
- **Topic-based subscriptions**: Subscribe to specific topics to receive targeted notifications
- **Cross-platform**: Works on Android, iOS (PWA), and Desktop browsers
- **Admin Panel**: Web-based admin interface for managing topics and subscriptions
- **Secure Authentication**: Bcrypt-hashed passwords and JWT token-based authentication
- **Rate Limiting**: Built-in brute-force protection for admin login
- **Topic Protection**: Optional bcrypt-hashed secret keys for topics
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

# Copy and configure environment variables
cp .env.example .env
nano .env  # Edit CORS_ORIGINS, VAPID_SUBJECT, etc.

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
  -e CORS_ORIGINS=https://your-domain.com \
  pushem sh -c "cd data && ../pushem"
```

Access the web interface at `http://localhost:8080`

The database and VAPID keys will be persisted in the `./data` directory.

### Important Configuration

**Before deploying to production**, edit `.env` and set:
- `CORS_ORIGINS` - Your domain(s) to restrict API access
- `VAPID_SUBJECT` - Your email for push notification authentication
- `MESSAGE_RETENTION_DAYS` - How long to keep message history (default: 7)
- `ADMIN_PASSWORD` - Strong password for admin panel access

Example `.env` for production:
```bash
CORS_ORIGINS=https://your-domain.com
VAPID_SUBJECT=mailto:admin@your-domain.com
MESSAGE_RETENTION_DAYS=7
ADMIN_PASSWORD=your-secure-admin-password-here
```

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
   # Docker Compose (with profiles)
   docker-compose --profile caddy up -d

   # Podman Compose (if profiles supported)
   podman-compose --profile caddy up -d

   # Older Podman: Uncomment Caddy service in docker-compose.yml
   # Edit docker-compose.yml and remove the "profiles:" section from caddy service
   nano docker-compose.yml
   podman-compose up -d
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
docker-compose logs -f caddy
```

### Container Management

```bash
# View logs
docker-compose logs -f
# Or for just one service:
docker-compose logs -f pushem
docker-compose logs -f caddy

# Stop the service
docker-compose down
# Or with Caddy:
docker-compose --profile caddy down

# Rebuild after code changes
docker-compose up -d --build

# Restart a specific service
docker-compose restart pushem
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

### Admin Panel

Access the admin panel at `http://localhost:8080/admin` to manage topics and subscriptions.

**Features:**
- View all topics with subscription and message counts
- Delete topics (removes all subscriptions and messages)
- Remove topic protection
- Secure JWT token-based authentication
- Rate limiting to prevent brute-force attacks

**First-time setup:**
1. Set `ADMIN_PASSWORD` in your `.env` file
2. Restart Pushem
3. Navigate to `/admin` and login with your password
4. Your session will remain active for 60 minutes (configurable)

**Security:**
- Password is hashed with bcrypt (never stored in plain text)
- JWT tokens expire after configurable duration (default: 60 minutes)
- Rate limiting: 5 failed attempts per 15 minutes (configurable)
- Password only transmitted once during login

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

#### `POST /topics/{topic}/protect`

Protect a topic with a secret key. Secrets are hashed with bcrypt before storage.

**Body:**
```json
{
  "secret": "your-secret-key-min-8-chars"
}
```

**Response:**
```json
{
  "status": "topic protected"
}
```

#### Admin API Endpoints

All admin endpoints require authentication with a JWT token obtained from the login endpoint.

**`POST /api/admin/login`**

Login to the admin panel and receive a JWT token.

**Body:**
```json
{
  "password": "your-admin-password"
}
```

**Response:**
```json
{
  "token": "eyJhbGc...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**`GET /api/admin/topics`**

List all topics with metadata (requires Bearer token in Authorization header).

**Response:**
```json
[
  {
    "name": "my-topic",
    "is_protected": true,
    "subscription_count": 5,
    "message_count": 42,
    "created_at": "2024-01-28T12:00:00Z"
  }
]
```

**`DELETE /api/admin/topics/{topic}`**

Delete a topic and all associated subscriptions and messages.

**`DELETE /api/admin/topics/{topic}/protection`**

Remove protection from a topic.

## Configuration

### Environment Variables

**Server Configuration:**
- `PORT`: Server port (default: 8080)
- `STATIC_DIR`: Path to frontend static files (default: web/dist)

**CORS & Security:**
- `CORS_ORIGINS`: Comma-separated list of allowed origins (default: localhost only)
  - Example: `https://your-domain.com`
  - Multiple: `https://domain1.com,https://domain2.com`
  - Public API: `https://*,http://*` (not recommended for private deployments)

**Web Push:**
- `VAPID_SUBJECT`: Email for VAPID authentication (default: mailto:admin@pushem.local)

**Message History:**
- `MESSAGE_RETENTION_DAYS`: Number of days to keep message history (default: 7)
- `CLEANUP_INTERVAL_HOURS`: Hours between automatic cleanup runs (default: 24)

**Admin Panel:**
- `ADMIN_PASSWORD`: Password for admin panel access (required, hashed with bcrypt)
- `ADMIN_TOKEN_EXPIRY_MINUTES`: JWT token expiration in minutes (default: 60)
- `ADMIN_MAX_LOGIN_ATTEMPTS`: Max failed login attempts before rate limiting (default: 5)
- `ADMIN_LOGIN_RATE_LIMIT_MINUTES`: Rate limit time window in minutes (default: 15)

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
  - Secrets are hashed with bcrypt before storage (never stored in plain text)
  - Provides defense in depth if database is compromised
  - **Allowed characters**: Letters, numbers, hyphens, underscores, dots (no special chars)
  ```bash
  # Protect a topic (min 8 chars, alphanumeric + -_.)
  curl -X POST http://localhost:8080/topics/my-topic/protect \
    -H "Content-Type: application/json" \
    -d '{"secret": "my-secure-key-2024"}'

  # Publish with authentication
  curl -X POST http://localhost:8080/publish/my-topic \
    -H "X-Pushem-Key: my-secure-key-2024" \
    -d "Protected message"
  ```

- **HTTPS Required**: For production deployments, HTTPS is mandatory for Service Workers
  - Use Caddy for automatic Let's Encrypt certificates (see CADDY_SETUP.md)

- **VAPID Keys**: Keep `vapid_keys.json` secure. Losing these keys will invalidate all subscriptions
  ```bash
  chmod 600 data/vapid_keys.json
  ```

- **Admin Panel Security**:
  - Bcrypt-hashed admin password (never stored in plain text)
  - JWT token-based authentication with configurable expiration
  - Built-in rate limiting (5 attempts per 15 minutes by default)
  - Session tokens stored in browser sessionStorage
  - Automatic logout on token expiration

- **Rate Limiting**: Built-in rate limiting for admin login; use Caddy or a reverse proxy for additional API rate limiting

For detailed security information, see:
- [SECURITY.md](SECURITY.md) - Security features and best practices
- [SECURITY_AUDIT.md](SECURITY_AUDIT.md) - Complete security audit report

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

### Admin API usage

```bash
# Login and get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"password":"your-admin-password"}' | jq -r '.token')

# List all topics
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/topics

# Delete a topic
curl -X DELETE \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/topics/my-topic
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
