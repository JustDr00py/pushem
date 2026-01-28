# Frontend Specification

## Tech Stack
- React (Vite)
- TypeScript
- TailwindCSS (Standard for rapid UI)

## Service Worker (`sw.js`)
The Service Worker is critical for handling background push events.

### `push` Event
- Decode the payload (JSON).
- Call `self.registration.showNotification(title, options)`.
- **Options**: `body`, `icon`, `data` (containing `click_url`).

### `notificationclick` Event
- Close the notification.
- Check if the app window is open:
    - If yes, focus it.
    - If no, open a new window.
- Navigate to `data.click_url` if present.

## PWA Manifest (`manifest.json`)
Required for "Add to Home Screen".
```json
{
  "name": "Pushem",
  "short_name": "Pushem",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#000000",
  "icons": [ ... ]
}
```

## UI Components

### 1. Subscription Manager (Main Page)
- **Input**: Topic Name (e.g., "my-alerts").
- **Button**: "Subscribe".
- **Logic**:
    1. Request Notification Permission.
    2. Get `PushSubscription` from browser (using VAPID public key from API).
    3. Send subscription object to `POST /subscribe/{topic}`.

### 2. iOS Instruction Modal
- **Trigger**: User clicks "Subscribe" on iOS (User-Agent check) AND `!window.navigator.standalone`.
- **Content**: "To receive notifications on iOS, tap the Share button and select 'Add to Home Screen'. You must run this as an app."

## Build Process
- `npm run build` -> outputs to `dist/`.
- Go backend embeds `dist/` or serves it from the filesystem.
