# Backend Specification

## Tech Stack
- **Language**: Go
- **Dependencies**:
    - `github.com/go-chi/chi/v5` (Router)
    - `github.com/mattn/go-sqlite3` (SQLite Driver)
    - `github.com/SherClockHolmes/webpush-go` (Web Push)
    - `github.com/go-chi/cors` (CORS handling)

## Database Schema (SQLite)

We need a single table to map topics to user subscriptions.

```sql
CREATE TABLE IF NOT EXISTS subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(topic, endpoint)
);
```

## API Endpoints

### 1. `GET /vapid-public-key`
Returns the VAPID Public Key so the frontend can use it to subscribe with the browser's Push Manager.
- **Response**: `{"publicKey": "..."}`

### 2. `POST /subscribe/{topic}`
Registers a device subscription for a specific topic.
- **Body**:
  ```json
  {
    "endpoint": "https://fcm.googleapis.com/fcm/send/...",
    "keys": {
      "p256dh": "...",
      "auth": "..."
    }
  }
  ```
- **Action**: Insert or Update into `subscriptions` table.

### 3. `POST /publish/{topic}`
Sends a notification to all subscribers of the topic.
- **Body**: Plain text string OR JSON:
  ```json
  {
    "title": "Alert",
    "message": "Something happened!",
    "click_url": "https://google.com"
  }
  ```
- **Logic**:
    1. Query `subscriptions` for `{topic}`.
    2. Iterate and send Web Push notification to each endpoint using `webpush-go`.
    3. Handle 410 Gone (unsubscribe invalid endpoints).

### 4. `GET /*`
Serves the React Frontend (Static Files).

## Project Structure
```text
Pushem/
├── cmd/
│   └── server/
│       └── main.go       # Entry point
├── internal/
│   ├── api/
│   │   └── handlers.go   # HTTP Handlers
│   ├── db/
│   │   └── sqlite.go     # DB init and queries
│   └── webpush/
│       └── service.go    # Push logic
├── web/                  # Frontend Source
├── go.mod
└── README.md
```
