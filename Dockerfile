# --- Build frontend ---
FROM node:22-bookworm-slim AS frontend
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

# --- Build backend ---
FROM golang:1.26-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -o /openilink-hub .

# --- Runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /openilink-hub /usr/local/bin/openilink-hub
EXPOSE 9800
ENTRYPOINT ["openilink-hub"]
CMD ["-listen", "0.0.0.0:9800"]
