# Implementation Steps

This guide is for the AI agent to execute the project.

## Phase 1: Project Skeleton & Backend Core
1.  Initialize Go module: `go mod init pushem`.
2.  Create directory structure (`cmd`, `internal`, `web`).
3.  Install Go dependencies: `chi`, `sqlite3`, `webpush-go`.
4.  Implement `internal/db`: SQLite setup and schema migration.
5.  Implement `internal/webpush`: VAPID key generation/loading.

## Phase 2: Frontend Implementation
1.  Initialize Vite React TS app in `web/` directory.
2.  Clean up default boilerplate.
3.  Create `public/manifest.json` and generate icons (placeholders).
4.  Create `public/sw.js` (Service Worker) in the public folder so it's served at root.
    *   *Note*: Ensure headers allow SW registration if serving from Go.
5.  Implement Main Page UI:
    *   Topic Input.
    *   Subscribe Logic (Permission -> Browser Subscription -> API Call).
    *   iOS Detection & Guide.

## Phase 3: Backend API Integration
1.  Implement `POST /subscribe/{topic}`: Save subscription to DB.
2.  Implement `POST /publish/{topic}`: Fetch subscriptions and send Push.
3.  Implement `GET /vapid-public-key`.
4.  Serve Frontend Statics from `web/dist`.

## Phase 4: Testing & Polish
1.  Verify "Add to Home Screen" flow.
2.  Test `curl` command for publishing.
3.  Ensure 410 Gone errors remove old subscriptions.
4.  Write README with setup instructions.
