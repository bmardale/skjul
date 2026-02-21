FROM --platform=$BUILDPLATFORM oven/bun:1 AS web-build
WORKDIR /src/web

COPY apps/web/package.json apps/web/bun.lock ./
RUN bun install --frozen-lockfile

COPY apps/web/ ./
RUN bun run build:docker

FROM --platform=$BUILDPLATFORM golang:1.25.7-alpine AS go-build
ARG TARGETARCH
WORKDIR /src

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

COPY --from=web-build /src/web/dist/ internal/static/dist/

ENV CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH
RUN go build -trimpath -ldflags="-s -w" -o /out/skjul ./cmd/skjul

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=go-build /out/skjul /skjul

EXPOSE 8080
ENTRYPOINT ["/skjul"]
