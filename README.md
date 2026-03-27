# URL Shortener

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A production-ready, horizontally scalable URL shortening service written in Go. It takes long URLs and squeezes them into short, shareable tokens — with support for custom redirect headers, Open Graph metadata for social media link previews, multi-node coordination, and full OpenTelemetry observability.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
  - [High-Level Overview](#high-level-overview)
  - [Project Structure](#project-structure)
  - [Key Design Decisions](#key-design-decisions)
- [How It Works](#how-it-works)
  - [URL Shortening Flow](#url-shortening-flow)
  - [Redirect Flow](#redirect-flow)
  - [Distributed ID Generation](#distributed-id-generation)
  - [Open Graph Metadata](#open-graph-metadata)
- [API Reference](#api-reference)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Database Setup](#database-setup)
  - [Configuration](#configuration)
  - [Build & Run](#build--run)
- [Configuration Reference](#configuration-reference)
- [Observability](#observability)
- [Development](#development)
- [Roadmap](#roadmap)
- [License](#license)

---

## Features

- [x] **URL Shortening** — Generate compact Base62 tokens from long URLs
- [x] **HTTP Redirect** — `GET /{token}` redirects to the original URL (`302 Found`)
- [x] **Custom Redirect Headers** — Attach arbitrary HTTP headers to redirect responses
- [x] **Open Graph Metadata** — Automatically fetches and caches OG tags from original URLs; serves rich link previews to social media bots (Facebook, Twitter, Slack, Discord, LinkedIn, Telegram, WhatsApp, Reddit, and more)
- [x] **CRUD REST API** — Create, fetch, and delete shortened URLs via a JSON API
- [x] **Multi-Node Deployment** — Distributed ID range coordination via MySQL, enabling horizontal scaling without token collisions
- [x] **MySQL Storage** — Persistent storage with configurable connection pooling
- [x] **OpenTelemetry Observability** — Distributed tracing and metrics (stdout or OTLP exporter) with trace-correlated structured JSON logs
- [x] **Health Check Endpoint** — Kubernetes-ready liveness/readiness probe
- [x] **TOML Configuration** — File-based configuration with environment variable override for the config path
- [x] **Graceful Shutdown** — Clean HTTP server and telemetry shutdown on `SIGTERM`/`SIGINT`
- [x] **Build Metadata** — Git commit hash and tag injected at compile time via `ldflags`

---

## Architecture

### High-Level Overview

```
┌──────────────┐       ┌──────────────────────────────┐       ┌───────────┐
│              │       │        URL Shortener          │       │           │
│   Clients    │──────▶│                               │──────▶│   MySQL   │
│  (Browser,   │ HTTP  │  ┌─────────┐  ┌───────────┐  │       │           │
│   Bot, API)  │◀──────│  │ Handler │─▶│  UseCase  │  │       │ url_token │
│              │       │  └─────────┘  └─────┬─────┘  │       │ coord tbl │
└──────────────┘       │                     │        │       └───────────┘
                       │  ┌──────────┐  ┌────▼─────┐  │
                       │  │ OG Fetch │  │ ID Mgr   │  │
                       │  └──────────┘  └──────────┘  │
                       │                               │
                       │  OpenTelemetry (traces+metrics)│
                       └──────────────────────────────┘
```

### Project Structure

```
.
├── cmd/server/                  # Application entry point
│   ├── main.go                  # Bootstrap & dependency wiring
│   └── config/                  # TOML configuration loading & validation
├── internal/
│   ├── domain/                  # Core domain models (URL, Range)
│   ├── id/                      # Distributed ID generation & range management
│   │   ├── manager.go           # Thread-safe sequential ID provider
│   │   ├── rangemanager.go      # Range manager interface
│   │   ├── rangemanager_datastore.go  # MySQL-backed range coordination
│   │   └── rangemanager_inmemory.go   # In-memory range manager (testing)
│   ├── token/                   # Base62 token encoding
│   ├── opengraph/               # OG metadata fetcher & bot detection
│   ├── usecase/                 # Business logic layer
│   │   ├── url/crud.go          # URL CRUD operations + async OG fetch
│   │   └── healthcheck.go       # Health check service
│   ├── storage/                 # Repository & coordinator interfaces
│   │   └── mysql/               # MySQL implementations
│   ├── server/protocol/http/    # HTTP handlers, routing & middleware
│   └── infrastructure/          # Cross-cutting concerns
│       ├── logger/              # Structured JSON logging with trace correlation
│       ├── telemetry/           # OpenTelemetry setup (tracing + metrics)
│       ├── errors/              # Domain error types
│       └── sql/                 # Database factory & DSN builder
├── docker/mysql/init.sql        # Database schema & triggers
├── config.toml.dist             # Example configuration file
├── Makefile                     # Build, test, lint commands
└── docs/                        # Additional documentation
```

### Key Design Decisions

| Concern | Approach |
|---|---|
| **Clean Architecture** | Strict separation into domain → use case → handler layers with dependency inversion through interfaces |
| **Distributed ID Generation** | Range-based coordination: each node reserves a contiguous block of IDs from MySQL, generating IDs locally without per-request DB calls |
| **Token Encoding** | Integer IDs are encoded to short Base62 strings (`[0-9a-zA-Z]`), producing compact, URL-safe tokens |
| **Conflict Resolution** | Token save retries up to 3 times on duplicate key collisions with newly generated IDs |
| **OG Metadata** | Fetched asynchronously after URL creation; served inline as HTML to detected social media bots |
| **Observability** | OpenTelemetry spans and metrics are embedded at every layer (handler → use case → repository → ID manager) |

---

## How It Works

### URL Shortening Flow

1. Client sends `POST /url` with the original URL (and optional custom headers).
2. The **ID Manager** provides the next sequential integer from its locally reserved range.
3. The integer is encoded into a **Base62 token** (e.g., `12345` → `dnh`).
4. The token + URL + headers are **persisted to MySQL**.
5. In a background goroutine, the service **fetches Open Graph metadata** from the original URL and stores the rendered HTML meta tags.
6. The token is returned to the client.

### Redirect Flow

1. A user or bot visits `GET /{token}`.
2. The service fetches the URL record from MySQL.
3. **Bot detection**: if the User-Agent matches a known crawler (Facebook, Twitter, Slack, etc.) _and_ OG metadata exists, an HTML page with OG meta tags and a `<meta http-equiv="refresh">` redirect is served — enabling rich link previews.
4. **Normal users**: a `302 Found` redirect is issued with any stored custom headers set on the response.

### Distributed ID Generation

To support horizontal scaling without a centralized ID service:

1. On startup, each node contacts the **coordination table** in MySQL to reserve a range of IDs (e.g., `[101, 200]`).
2. IDs are generated locally from the reserved range with a thread-safe counter.
3. When the range is exhausted, a new range is atomically reserved using **optimistic concurrency control** (version column + MySQL trigger to prevent stale writes).
4. A **journal table** records every range allocation for auditability.

```
Node A reserves [1, 100]      → generates tokens from IDs 1–100
Node B reserves [101, 200]    → generates tokens from IDs 101–200
Node A exhausts range         → reserves [201, 300]
```

### Open Graph Metadata

When a shortened URL is shared on social media, platform crawlers look for Open Graph `<meta>` tags to render a rich preview (title, description, image). This service:

1. **Fetches** OG tags from the original page asynchronously after URL creation.
2. **Caches** the rendered HTML meta tags in the `og_html` column.
3. **Serves** a full HTML page with OG tags to detected bots, followed by an automatic redirect to the original URL.
4. Supports **manual refresh** via `PUT /url/{token}/og` to re-fetch stale metadata.

Detected bots include: Facebook, Twitter, Slack, LinkedIn, WhatsApp, Telegram, Discord, Pinterest, Reddit, Google, Bing, Apple, and more.

---

## API Reference

### Shorten a URL

```http
POST /url
Content-Type: application/json

{
  "url": "https://example.com/very/long/path",
  "headers": {
    "X-Custom-Header": "value"
  }
}
```

**Response** `200 OK`

```json
{
  "token": "dnh"
}
```

### Fetch URL Details

```http
GET /url/{token}
```

**Response** `200 OK`

```json
{
  "url": "https://example.com/very/long/path",
  "token": "dnh",
  "headers": {
    "X-Custom-Header": "value"
  }
}
```

### Delete a Shortened URL

```http
DELETE /url/{token}
```

**Response** `202 Accepted`

### Refresh Open Graph Metadata

```http
PUT /url/{token}/og
```

**Response** `202 Accepted`

### Redirect

```http
GET /{token}
```

**Response** `302 Found` with `Location` header pointing to the original URL (custom headers included). Bots receive `200 OK` with an HTML page containing OG meta tags.

### Health Check

```http
GET /healthcheck
```

---

## Getting Started

### Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/dl/)
- **MySQL 5.7+** (or MariaDB 10.3+)
- **Make** (optional, for convenience commands)

### Database Setup

1. Create the database:

```sql
CREATE DATABASE url_shortener CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. Initialize the schema by running the provided SQL script:

```bash
mysql -u root -p url_shortener < docker/mysql/init.sql
```

This creates three tables:

| Table | Purpose |
|---|---|
| `url_token` | Stores shortened URLs, custom headers (JSON), and cached OG HTML |
| `nodes_coordination_keys` | Tracks the global last-reserved ID with optimistic versioning |
| `node_range_journal` | Audit log of every ID range allocation per node |

### Configuration

Copy the example configuration and adjust values:

```bash
cp config.toml.dist config.toml
```

Optionally set a custom config path:

```bash
export CONFIG_FILE_PATH=/path/to/config.toml
```

If `CONFIG_FILE_PATH` is not set, the service defaults to `./config.toml`.

See [Configuration Reference](#configuration-reference) for all available options.

### Build & Run

```bash
# Build the binary (git tag + commit hash injected automatically)
make build

# Run the server
make run

# Or build and run manually
go build -ldflags "-X main.CommitHash=$(git rev-parse --short HEAD) -X main.Tag=$(git describe --tags --always)" -o bin/url-shortener ./cmd/server
./bin/url-shortener
```

The server starts on the configured port (default `:8080`).

**Quick test:**

```bash
# Shorten a URL
curl -X POST http://localhost:8080/url \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/majidgolshadi/url-shortner"}'

# Redirect (in a browser or with curl -L)
curl -L http://localhost:8080/{token}
```

---

## Configuration Reference

The configuration is loaded from a TOML file. Below is the full reference with defaults from `config.toml.dist`:

```toml
environment = "development"          # development | staging | production
http_addr = "8080"                   # HTTP listen port
service_name = "url-shortener"       # Service name for logging & telemetry
log_level = "info"                   # trace | debug | info | warn | error | fatal
enable_application_profiler = false

[app_db]                             # Application database (URL storage)
max_open_conn = 20
read_timeout_sec = 1
write_timeout_sec = 1
connection_lifetime_sec = 10

[app_db.credential]
host = "127.0.0.1:3306"
db_name = "url_shortener"
username = "root"
password = "toor"

[coordination]                       # Distributed ID coordination
node_id = "node_1"                   # Unique node identifier within the cluster
range_size = 100                     # Number of IDs to reserve per allocation

[coordination.datastore]             # Coordination database (can be same as app_db)
max_open_conn = 20
read_timeout_sec = 1
write_timeout_sec = 1
connection_lifetime_sec = 10

[coordination.datastore.credential]
host = "127.0.0.1:3306"
db_name = "url_shortener"
username = "root"
password = "toor"

[telemetry]
enabled = true                       # Enable OpenTelemetry tracing & metrics
exporter_type = "stdout"             # "stdout" for dev, "otlp" for production
otlp_endpoint = "localhost:4318"     # OTLP HTTP collector endpoint

[opengraph]
fetch_timeout_sec = 10               # Timeout for fetching OG metadata from original URLs
```

> **Multi-node deployment**: Each node must have a unique `coordination.node_id`. The `range_size` controls how many IDs each node reserves at a time — larger values reduce coordination overhead but may leave gaps if a node restarts.

---

## Observability

The service is fully instrumented with [OpenTelemetry](https://opentelemetry.io/):

### Tracing

Distributed traces propagate through every layer:

- **HTTP middleware** (`otelmux`) — automatic span creation for every request
- **Use case layer** — `Service.Add`, `Service.Fetch`, `Service.Delete`, `Service.RefreshOG`
- **Repository layer** — individual spans for each MySQL query
- **ID Manager** — spans for ID generation and range acquisition

### Metrics

| Metric | Type | Description |
|---|---|---|
| `url.add.total` | Counter | Total URL shorten operations |
| `url.add.errors` | Counter | Total URL shorten errors |
| `url.add.duration_ms` | Histogram | Shorten operation latency |
| `url.fetch.total` | Counter | Total URL fetch operations |
| `url.fetch.errors` | Counter | Total URL fetch errors |
| `url.fetch.duration_ms` | Histogram | Fetch operation latency |
| `url.delete.total` | Counter | Total URL delete operations |
| `url.delete.errors` | Counter | Total URL delete errors |
| `url.delete.duration_ms` | Histogram | Delete operation latency |
| `id.generated.total` | Counter | Total IDs generated |
| `id.generate.duration_ms` | Histogram | ID generation latency |
| `id.range.remaining` | UpDownCounter | Remaining IDs in current range |
| `db.query.duration_ms` | Histogram | MySQL query latency (by operation) |
| `db.query.errors` | Counter | MySQL query errors (by operation) |
| `db.coordinator.query.duration_ms` | Histogram | Coordinator query latency |
| `db.coordinator.query.errors` | Counter | Coordinator query errors |

### Structured Logging

Logs are emitted as **JSON** via [Logrus](https://github.com/sirupsen/logrus) with automatic **trace correlation** — every log entry within a traced request includes `trace_id` and `span_id` fields, enabling seamless log-to-trace navigation in backends like Grafana, Jaeger, or Datadog.

```json
{
  "timestamp": "2026-03-27T12:00:00.000",
  "level": "info",
  "message": "URL shortened successfully",
  "component": "url_service",
  "token": "dnh",
  "url": "https://example.com",
  "trace_id": "abc123...",
  "span_id": "def456..."
}
```

### Exporter Modes

| Mode | `exporter_type` | Use Case |
|---|---|---|
| **stdout** | `"stdout"` | Local development — traces and metrics printed to console |
| **OTLP** | `"otlp"` | Production — sends data to an OpenTelemetry Collector via HTTP (`otlp_endpoint`) |

---

## Development

### Available Make Targets

```bash
make build          # Build the binary to bin/url-shortener
make run            # Build and run
make test           # Run all tests with race detection
make test-verbose   # Run tests with verbose output
make lint           # Run golangci-lint
make fmt            # Format Go source files (gofmt + goimports)
make vet            # Run go vet
make clean          # Remove build artifacts
make help           # Show all available targets
```

### Running Tests

```bash
# All tests
make test

# Specific package
go test -race -count=1 ./internal/usecase/url/...
go test -race -count=1 ./internal/id/...
go test -race -count=1 ./internal/opengraph/...
go test -race -count=1 ./internal/server/protocol/http/...
```

### Code Quality

```bash
# Lint
make lint

# Format
make fmt

# Vet
make vet
```

---

## Roadmap

- [ ] Collect access statistics (click counts, referrers, geo)
- [ ] Customer registration and authentication
- [ ] Per-customer token counter ranges
- [ ] Rate limiting
- [ ] Caching layer (Redis) for hot URLs
- [ ] OpenAPI (Swagger) specification

---

## License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

Copyright (c) 2018 Majid Golshadi
