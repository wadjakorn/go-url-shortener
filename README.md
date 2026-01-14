# Go URL Shortener

A high-performance URL shortener written in Go, originally using SQLite (Hexagonal Architecture) with support for easy migration to PostgreSQL.

## Features
-   Shorten URLs with random or custom codes
-   Soft-delete links
-   List links with pagination, search, and tag filtering
-   Tag support (stored as JSON)
-   **Hexagonal Architecture**: Core logic is independent of infrastructure.
-   **Import/Export CLI**: Easily migrate data to/from JSON.

## Getting Started

### Prerequisites
-   Go 1.22+ (or use Docker)

### Installation
1.  Clone the repository
2.  Copy `.env.example` to `.env`
    ```bash
    cp .env.example .env
    ```
3.  Run the server
    ```bash
    go run cmd/server/main.go
    ```

### Usage

**Create Link**
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://google.com", "title": "Google", "tags": ["search", "tech"]}'
```

**List Links**
```bash
curl "http://localhost:8080/api/v1/links?page=1&limit=10"
```


**Get/Redirect**
Open `http://localhost:8080/{short_code}` in your browser.

**Usage Stats**
Get detailed stats for a specific link (total clicks, referrers, daily history):
```bash
curl "http://localhost:8080/api/v1/links/{id}/stats"
```

**Dashboard**
Get ranking of top 10 links by clicks and total system usage.
Supports filters: `tag`, `search` (title), `domain` (original url).
```bash
curl "http://localhost:8080/api/v1/dashboard"
curl "http://localhost:8080/api/v1/dashboard?tag=tech&limit=5"
```

### Data Migration (CLI)

Export data to JSON (backup or migration):
```bash
go run cmd/cli/main.go export > backup.json
```

Import data from JSON:
```bash
go run cmd/cli/main.go import --file=backup.json
```

### Deployment

**Docker**
```bash
docker build -t shortener .
docker run -p 8080:8080 shortener
```

**Vercel**
Deploy directly with Vercel CLI. The project is configured with `vercel.json` to use Go Serverless functions.
*Note*: SQLite on Vercel is ephemeral. Use a cloud database (Turso/Neon) for persistence.
