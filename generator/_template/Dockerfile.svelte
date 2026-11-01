# Web stage:
# Build the SvelteKit app with the pnpm version pinned in web/package.json.
FROM node:24-alpine AS web
WORKDIR /app/web

RUN npm install --global pnpm@12.9.1

# Copy the package files first so Docker can cache the install.
COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./
RUN pnpm install --frozen-lockfile

# Copy the app and build it into web/build.
COPY web/ ./
RUN pnpm build

# Build stage:
# Build the server as a static binary with the app embedded.
FROM golang:1.26 AS build
WORKDIR /app

# Copy Go dependency files first so Docker can cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project and the built app.
COPY . .
COPY --from=web /app/web/build ./web/build

# Build the application as a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

# Deploy-Stage
FROM alpine:3.24
WORKDIR /app

# Install ca-certificates and sqlite for backups/debugging
RUN apk add --no-cache ca-certificates sqlite

# Set environment variable for runtime
ENV APP_ENV=production
ENV GO_ENV=production

# Copy only the binary. It embeds the SvelteKit app.
COPY --from=build /app/main .

# Expose the port
EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8090/healthz >/dev/null || exit 1

# Command to run
CMD ["./main"]
