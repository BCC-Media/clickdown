# ClickDown

A dense, keyboard-driven local task list backed by your ClickUp data.

Pulls every task assigned to you (across all teams, not closed) into a local SQLite mirror, lets you triage from a fast Vue UI, and pushes title/description/status changes back to ClickUp on the next sync. Single self-contained Go binary — the SPA is embedded.

## Features

- **Single binary**, no runtime deps. SPA + DB migrations embedded.
- **Sync**: on startup, every 10 minutes (configurable), and on demand via `Sync now` or `POST /api/sync`.
- **Scope**: tasks assigned to the authenticated user, across all teams (workspaces) the token can see. Closed tasks are not fetched.
- **Push back**: only `title`, `description`, `status` (per design). Priority and tags from ClickUp are read-only locally; priority can't be pushed back.
- **Local-only tags**: add tags that don't exist in ClickUp — they persist locally and never leak upstream.
- **Verbatim ClickUp statuses**: status pills, filter chips, and the `1`–`9` keyboard shortcuts are driven by your list's actual status names and colors.
- **Open in ClickUp**: `o` on a focused row (or click the `open ↗` link) opens the task at `app.clickup.com/t/{id}` in a new tab.

## Quick start

```bash
# 1. Drop your token in a .env file
echo 'CLICKUP_API_TOKEN=pk_xxx_...' > .env

# 2. Build and run
make build
./dist/clickdown
# → http://127.0.0.1:7878
```

Get a personal token at <https://app.clickup.com/settings/apps>.

## Make targets

| Target            | What it does                                                |
|-------------------|-------------------------------------------------------------|
| `make build`      | Build the SPA + binary for the host platform                |
| `make build-mac`  | Build `darwin/arm64` and `darwin/amd64` binaries            |
| `make build-windows` | Build `windows/amd64` binary                             |
| `make build-all`  | All of the above                                            |
| `make run`        | Build and run                                               |
| `make dev`        | Run Go API on `:7878` + Vite dev server on `:5173` together |
| `make dev-api`    | Just the Go API                                             |
| `make dev-web`    | Just the Vite dev server (proxies `/api` to `:7878`)        |

## Configuration

Resolved in order: flags → env → `.env` file → defaults.

`.env` is auto-loaded from the working directory and from `~/.clickdown/.env` (existing process env always wins).

| Flag             | Env                       | Default                       | Purpose                          |
|------------------|---------------------------|-------------------------------|----------------------------------|
| `-token`         | `CLICKUP_API_TOKEN`       | _(required for sync)_         | ClickUp personal API token       |
| `-db`            | `CLICKDOWN_DB`            | `~/.clickdown/clickdown.db`   | SQLite file path                 |
| `-addr`          | `CLICKDOWN_ADDR`          | `127.0.0.1:7878`              | HTTP listen address              |
| `-sync-interval` | `CLICKDOWN_SYNC_INTERVAL` | `10m`                         | Background sync interval         |

The interval is also editable at runtime via the Tweaks panel (gear icon, bottom-right).

## Keyboard

| Key                 | Action                                       |
|---------------------|----------------------------------------------|
| `j` / `k` / `↑` / `↓` | Move focus                                 |
| `↵`                 | Expand/collapse the focused row              |
| `e`                 | Edit title of the focused row                |
| `1`–`9`             | Set status of the focused row (by orderindex)|
| `o`                 | Open the focused task in ClickUp (new tab)   |
| `/`                 | Focus search                                 |
| `x`                 | Toggle selection                             |
| `esc`               | Close expanded row / clear search            |

## Stack

- **Go 1.26+**, `modernc.org/sqlite` (pure Go), `pressly/goose/v3` for migrations, `sqlc` for query codegen.
- **Vue 3.5 + Vite 6**, vanilla CSS (Geist + Geist Mono from Google Fonts).
- SPA built into `web/dist/`, embedded into the binary via `//go:embed all:web/dist`.

## Repo layout

```
.
├── main.go                       # wires config → db → sync → http
├── Makefile
├── sqlc.yaml
├── internal/
│   ├── config/                   # env + flags + .env loader
│   ├── clickup/                  # REST client
│   ├── db/
│   │   ├── migrations/           # goose SQL migrations (embedded)
│   │   ├── queries/              # sqlc input
│   │   └── gen/                  # sqlc output (committed)
│   ├── server/                   # HTTP API + SPA serving
│   └── sync/                     # pull + push reconciliation worker
└── web/
    ├── index.html
    ├── vite.config.ts
    └── src/
        ├── App.vue
        ├── api.ts
        ├── components/           # Row, StatusPill, TagList, EditableTitle, PriorityBadge, TweaksPanel
        └── assets/styles.css
```

## Notes & design choices

- The local DB is a mirror; tasks `clickup_id` is canonical. Every fetched task is upserted; tasks that disappear from the remote response are soft-deleted (`deleted_at`).
- Conflict resolution is per-field: if the local field is dirty (in `task_dirty`), the pull skips that field; otherwise the remote wins when `remote.date_updated > local.clickup_updated_at`.
- Local mutations push on the next sync via the `task_dirty` queue, which survives restarts.
- Statuses are stored verbatim (no fixed 5-state model); the `statuses` table is refreshed every sync from each task's embedded status object.
- "Completed" statuses are hidden by default — both ClickUp's `type='closed'` and common name patterns (`published`, `archived`, `done`, `completed`, `cancel(l)ed`). Your filter tweaks persist in `localStorage`.
- `CLICKUP_TOKEN` is also accepted (legacy) but `CLICKUP_API_TOKEN` is preferred.
