# Birth Story — App Spec

## Overview

A simple Go web app that serves a live-updating birth story page. You send updates
from Discord using a slash command, and visitors see them appear in real time.

---

## Architecture

```
Discord (you)
    │  /update <message>
    ▼
Discord Bot (Go)
    │  HTTP POST /internal/update
    ▼
Go Web Server
    ├── GET /          → HTML page (birth story feed)
    └── GET /events    → SSE stream (live updates pushed to browser)
```

Both the bot listener and web server run in the same Go binary.

---

## Features

### Web Page (`GET /`)
- Full-screen, mobile-friendly HTML page served by Go
- Shows a chronological feed of updates (newest at top)
- Auto-connects to SSE stream — updates appear without a page refresh
- Minimal, clean design: baby name/due date in a header, feed below
- No login required (public URL, shareable with family)

### SSE Stream (`GET /events`)
- Server-Sent Events endpoint
- Each connected browser is a subscriber
- When a new update arrives, it is broadcast to all open connections
- Includes the update text and a timestamp

### Discord Slash Command (`/update`)
- Bot registers a `/update <message>` slash command on startup
- When you run it, the bot POSTs the message to the internal HTTP endpoint
- Bot replies ephemerally so your channel stays clean
- A shared secret (`INTERNAL_SECRET`) prevents unauthorized posts

### Storage
- Updates are stored in memory (a Go slice) for simplicity — no database needed
- On restart, history is lost (acceptable for a one-day birth story event)
- Optional: persist to a flat JSON file via `UPDATE_FILE` env var so restarts
  don't clear the feed

---

## Configuration (Environment Variables)

| Variable           | Required | Description                                      |
|--------------------|----------|--------------------------------------------------|
| `DISCORD_TOKEN`    | Yes      | Bot token from Discord Developer Portal          |
| `DISCORD_APP_ID`   | Yes      | Application ID from Discord Developer Portal     |
| `DISCORD_GUILD_ID` | Yes      | Your server's Guild ID (for instant cmd registration) |
| `INTERNAL_SECRET`  | Yes      | Shared secret between bot handler and web server |
| `PORT`             | No       | Web server port (default: `8080`)                |
| `UPDATE_FILE`      | No       | Path to persist updates as JSON (e.g. `/data/updates.json`) |

---

## File Structure

```
birth-story/
├── main.go              # Entry point — starts web server + registers Discord bot
├── server.go            # HTTP handlers: /, /events, /internal/update
├── bot.go               # Discord slash command registration + interaction handler
├── store.go             # In-memory (+ optional file) update store
├── templates/
│   └── index.html       # The birth story page (embedded via go:embed)
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

---

## Dockerfile

- Multi-stage build: `golang:1.23-alpine` builder → `alpine` final image
- Final image is ~15 MB
- Listens on `PORT` (default 8080)
- If `UPDATE_FILE` is set, mount a volume at that path for persistence

---

## Docker Compose (self-hosted VPS)

```yaml
services:
  birth-story:
    build: .
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file: .env
    volumes:
      - ./data:/data   # only needed if UPDATE_FILE=/data/updates.json
```

---

## Page Design (rough)

```
┌─────────────────────────────────────────┐
│  🍼  Baby [Last Name] is on the way!    │  ← centered header
│      March 18, 2026                     │
├─────────────────────────────────────────┤
│  ● 14:32  She's at 8cm!                 │  ← newest first
│  ● 13:10  Epidural in, resting now      │
│  ● 11:45  We're at the hospital!        │
└─────────────────────────────────────────┘
```

Soft pastel background, large readable font, works well on phones.

---

## Out of Scope

- Authentication / admin UI
- Photo uploads
- Multiple baby / multi-user support
- Push notifications

---

## Build Instructions (for Claude)

1. Implement `store.go` — thread-safe slice with optional JSON persistence
2. Implement `server.go` — SSE broadcaster, index handler, internal update handler
3. Implement `bot.go` — register `/update` slash command, handle interaction via HTTP
4. Implement `main.go` — wire everything together, read env vars
5. Write `templates/index.html` — embedded HTML with SSE JS client
6. Write `Dockerfile` (multi-stage) and `docker-compose.yml`
7. Write `.env.example`
