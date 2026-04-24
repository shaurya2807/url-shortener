# Scalable URL Shortener Service

High-performance URL shortener with Redis caching built in Go.

## Architecture

```
Client
  │
  ▼
app (REST API :8081)
  │
  ├──▶ PostgreSQL  (store URLs, click counts)
  │
  └──▶ Redis       (cache redirects, 24hr TTL)
```

## Tech Stack

| Layer    | Technology                   |
|----------|------------------------------|
| Language | Go 1.26                      |
| Framework| Gin                          |
| Database | PostgreSQL 17                |
| Cache    | Redis (alpine)               |
| Runtime  | Docker / Docker Compose      |

## How to Run

```bash
docker compose up
```

That's it. Compose builds the app image, starts Postgres and Redis with health checks, and waits for both to be ready before launching the app.

### Example Requests

**Shorten a URL**
```bash
curl -s -X POST http://localhost:8081/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com/very/long/path"}'
```
```json
{
  "short_code": "aB3kR9",
  "short_url": "http://localhost:8081/aB3kR9",
  "original_url": "https://example.com/very/long/path",
  "created_at": "2026-04-23T10:00:00Z"
}
```

**Redirect**
```bash
curl -L http://localhost:8081/aB3kR9
# → 301 redirect to https://example.com/very/long/path
```

**Get Stats**
```bash
curl -s http://localhost:8081/aB3kR9/stats
```
```json
{
  "short_code": "aB3kR9",
  "short_url": "http://localhost:8081/aB3kR9",
  "original_url": "https://example.com/very/long/path",
  "click_count": 42,
  "created_at": "2026-04-23T10:00:00Z"
}
```

## API Reference

| Method | Path              | Description                          |
|--------|-------------------|--------------------------------------|
| POST   | `/shorten`        | Create a short URL                   |
| GET    | `/:code`          | Redirect to original URL (301)       |
| GET    | `/:code/stats`    | Return click count and metadata      |

## Project Structure

```
.
├── cmd/server/main.go          # Entry point, wiring
├── configs/config.go           # Env-based configuration
├── internal/
│   ├── cache/redis.go          # Redis get/set with 24hr TTL
│   ├── handler/url_handler.go  # HTTP handlers (Gin)
│   ├── model/url.go            # Domain types and request/response structs
│   ├── repository/             # PostgreSQL queries via pgx
│   └── service/url_service.go  # Business logic
├── pkg/logger/logger.go        # Zap logger (dev vs production)
├── migrations/001_create_urls.sql
├── init/init.sql               # Bootstrapped by Compose on first run
├── Dockerfile
└── docker-compose.yml
```

## Key Engineering Decisions

**Redis cache for redirects** — The redirect path (`GET /:code`) is the hot path. Serving it from Redis avoids a round-trip to Postgres on every click. The 24-hour TTL balances freshness against cache efficiency; short codes are immutable once created, so stale reads are not a concern.

**`crypto/rand` for short codes** — The 6-character code is drawn from a 62-character alphabet using `crypto/rand`, giving ~56 billion unique codes with no predictable pattern. This prevents enumeration attacks that would expose all shortened URLs.

**Async click counting** — On a cache miss the redirect still returns immediately; click-count increments and cache writes are fired in background goroutines. This keeps p99 redirect latency decoupled from write latency under load.
