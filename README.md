# skjul

**skjul** /ʂʏlː/ — Norwegian for *hide*, *conceal*, or *secret place*

An end-to-end encrypted pastebin. Self-hostable. Zero knowledge.

## Features

- **True E2E encryption** — Server never sees plaintext; keys stay in your browser
- **Encrypted attachments** — Upload files to S3 with client-side encryption
- **Burn after reading** — One-time notes that delete after first view
- **Expiry** — Auto-delete after 30min to 30 days, or never
- **Syntax highlighting** — 20+ languages with theme-aware Prism
- **Share via URL** — Key in fragment (#key=...) never hits the server

## Quick Start

```bash
# With Docker Compose
docker-compose up -d

# Or build the single binary
cd apps/web && bun install && bun run build
cd ../api && go build -o skjul ./cmd/skjul
./skjul
```

## Tech

- **Frontend:** React + TypeScript + Vite + Tailwind
- **Backend:** Go + Gin + PostgreSQL
- **Crypto:** XChaCha20-Poly1305, Argon2id
- **Storage:** S3-compatible (optional)

## License

MIT
