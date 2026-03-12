## Cursor Cloud specific instructions

**skjul** is a zero-knowledge, end-to-end encrypted pastebin with a Go backend (`apps/api`) and React/Vite frontend (`apps/web`).

### Services overview

| Service | Path | How to run | Port |
|---------|------|-----------|------|
| Go API | `apps/api` | `go run ./cmd/skjul` (from `apps/api/`) | 8080 |
| Vite dev server | `apps/web` | `bun run dev` | 5173 |
| PostgreSQL 18 | system | `sudo pg_ctlcluster 18 main start` | 5433 |

### Key caveats

- **PostgreSQL 18 required.** The migrations use `uuidv7()`, which is only available in PostgreSQL 18+. The cluster runs on **port 5433** (not the default 5432).
- **Go embed requires `dist/` directory.** The Go backend embeds the frontend from `apps/api/internal/static/dist/`. Before compiling the backend, you must build the frontend (`bun run build` in `apps/web`) and copy `apps/web/dist` to `apps/api/internal/static/dist/`. Without this, `go build` fails.
- **Frontend API base URL.** In dev mode, create `apps/web/.env.local` with `VITE_API_BASE_URL=http://localhost:8080/` so the frontend `ky` HTTP client talks to the Go API. The backend has CORS configured for `http://localhost:5173`.
- **S3 is optional.** Leave `s3.bucket` and `s3.region` empty in `config.yaml` to skip S3 initialization; attachment uploads won't work but core paste functionality is fine.
- **Config location.** The Go backend reads `config.yaml` from the current working directory (or `./config/`). A dev config lives at `apps/api/config.yaml` — make sure to `cd apps/api` before running the server.
- **ESLint.** The codebase has pre-existing lint errors (`bun run lint` in `apps/web`). These are known issues in the existing code.

### Standard commands

- **Go tests:** `cd apps/api && go test ./...`
- **Frontend lint:** `cd apps/web && bun run lint`
- **Frontend build:** `cd apps/web && bun run build`
- **Go build:** `cd apps/api && go build -o skjul ./cmd/skjul` (after copying `dist/`)
