FROM oven/bun:1 AS web-build
WORKDIR /src/web

COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile

COPY web/ ./
RUN bun run build

FROM golang:1.25.7-alpine AS go-build
WORKDIR /src

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./

COPY --from=web-build /src/web/dist/ internal/static/dist/

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/skjul ./cmd/skjul

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 skjul

WORKDIR /
COPY --from=go-build /out/skjul /skjul

EXPOSE 8080
USER skjul
ENTRYPOINT ["/skjul"]
