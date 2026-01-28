# System Overview: Pushem

Pushem is a self-hosted notification service designed to be valid alternative to ntfy.sh. It allows users to send notifications via a simple REST API and receive them as Web Push notifications on their devices (Android, iOS PWA, Desktop).

## Architecture

```mermaid
graph TD
    User[User / Script] -->|POST /publish/:topic| API[Go Backend]
    API -->|Fetch Subscribers| DB[(SQLite)]
    API -->|Web Push Protocol| PushService[Push Service (FCM/Apple/Mozilla)]
    PushService -->|Deliver| Device[User Device (PWA)]
    Device -->|Subscribe| API
```

## Core Requirements

### 1. Backend (Go)
- **Language**: Go (Golang)
- **Web Framework**: `chi` or `net/http` (standard library with a lightweight router recommended).
- **Database**: SQLite (embedded, zero-config).
- **Push Protocol**: Web Push (VAPID authenticated).
- **Security**: HTTPS is required for Service Workers. (In development, localhost is exempt).

### 2. Frontend (PWA)
- **Framework**: React (Vite) + TypeScript.
- **PWA Features**: `manifest.json`, Service Worker (`sw.js`).
- **iOS Support**: Special handling for iOS "Add to Home Screen" instructions.

## Key Technical Decisions
- **No WebSockets**: As requested, exclusively utilize HTTP/2 Web Push.
- **VAPID**: Keys generated on first run and stored/logged.
- **Encryption**: AES-128-GCM standard for Web Push payloads.
