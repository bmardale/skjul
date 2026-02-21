# skjul

**skjul** /skjuːˀl/ — Danish for *hide*, *conceal*, or *secret place*

Zero-knowledge, end-to-end encrypted pastebin. Self-hostable.

> ⚠️ **Security Disclaimer**: The cryptographic implementation has not been professionally audited. Use at your own risk for sensitive data.

## Features

- **End-to-end encryption** — Server never sees plaintext; keys stay in your browser
- **Encrypted attachments** — Client-side encrypted file uploads to S3
- **Burn after reading** — Single-view notes that self-destruct
- **Expiry** — 30 minutes to 30 days, or indefinite
- **Syntax highlighting** — 20+ languages, theme-aware
- **Share via URL** — Keys in fragment (`#key=...`) never touch the server

## Quick Start

### Docker (Recommended)

```bash
# Pull and run from GitHub Container Registry
docker-compose up -d
```

Edit `postgres` password in `docker-compose.yml`, then start:
- Frontend: http://localhost:8080
- Requires: PostgreSQL configured in `config.yaml`

### Build from Source

```bash
# Build frontend
cd apps/web && bun install && bun run build

# Copy frontend dist to backend static assets
cp -r dist ../api/internal/static/

# Build backend (embeds frontend)
cd ../api && go build -o skjul ./cmd/skjul
./skjul
```

## Stack

- **Frontend:** React, TypeScript, shadcn/ui
- **Backend:** Go, Gin, PostgreSQL
- **Crypto:** XChaCha20-Poly1305, Argon2id
- **Storage:** S3-compatible (optional)
