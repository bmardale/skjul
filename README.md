# skjul

**skjul** /skjuːˀl/ — Danish for *hide*, *conceal*, or *secret place*

Zero-knowledge, end-to-end encrypted pastebin. Self-hostable.

## Features

- **End-to-end encryption** — Server never sees plaintext; keys stay in your browser
- **Encrypted attachments** — Client-side encrypted file uploads to S3
- **Burn after reading** — Single-view notes that self-destruct
- **Expiry** — 30 minutes to 30 days, or indefinite
- **Syntax highlighting** — 20+ languages, theme-aware
- **Share via URL** — Keys in fragment (`#key=...`) never touch the server

## Quick Start

```bash
# Docker Compose
docker-compose up -d

# Single binary
cd apps/web && bun install && bun run build
cd ../api && go build -o skjul ./cmd/skjul
./skjul
```

## Stack

- **Frontend:** React, TypeScript, shadcn/ui
- **Backend:** Go, Gin, PostgreSQL
- **Crypto:** XChaCha20-Poly1305, Argon2id
- **Storage:** S3-compatible (optional)
