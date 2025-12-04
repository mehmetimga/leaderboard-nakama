# Leaderboard Service with Nakama

A production-ready leaderboard service built with Go and [Nakama](https://heroiclabs.com/nakama/), featuring real-time rankings and periodic snapshots for official leaderboards.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Leaderboard API                          │
│                         (Go + Chi)                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────────┐   │
│  │   HTTP      │    │  Leaderboard │    │   Background    │   │
│  │  Handlers   │───▶│   Service    │◀───│    Worker       │   │
│  └─────────────┘    └──────────────┘    └─────────────────┘   │
│                            │                     │             │
│              ┌─────────────┴─────────────┐      │             │
│              ▼                           ▼      ▼             │
│       ┌────────────┐              ┌────────────────┐          │
│       │   Nakama   │              │   PostgreSQL   │          │
│       │  (Live LB) │              │ (Official LB)  │          │
│       └────────────┘              └────────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

- **Live Leaderboard**: Real-time rankings via Nakama's built-in leaderboard system
- **Official Leaderboard**: Periodic snapshots stored in PostgreSQL (configurable interval, default 30 min)
- **Background Worker**: Automatically snapshots live leaderboards to PostgreSQL

## Features

- ✅ Submit scores to leaderboards
- ✅ Fetch live (real-time) leaderboards from Nakama
- ✅ Fetch official (snapshot) leaderboards from PostgreSQL
- ✅ Get user rankings (live & official)
- ✅ Get records around a specific user
- ✅ Background worker for periodic snapshots
- ✅ Docker Compose setup for local development
- ✅ Clean architecture with interfaces for testing

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local development)

### Run with Docker Compose

```bash
# Start all services (Nakama, PostgreSQL, API)
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down
```

Services will be available at:
- **Leaderboard API**: http://localhost:8080
- **Nakama Console**: http://localhost:7351 (admin/password)
- **Nakama HTTP API**: http://localhost:7350
- **PostgreSQL**: localhost:5432

### Local Development

```bash
# Copy environment file
cp env.example .env

# Start dependencies only
docker-compose up -d postgres nakama

# Run the API locally
go run ./cmd/api

# Run tests
go test ./...
```

## API Reference

### Submit Score

```bash
POST /api/v1/scores
Content-Type: application/json

{
  "leaderboard_id": "global_scores",
  "user_id": "user-123",
  "username": "player1",
  "score": 1000,
  "subscore": 50,
  "metadata": {
    "level": "5"
  }
}
```

### Get Leaderboard

```bash
# Live leaderboard (from Nakama)
GET /api/v1/leaderboards/{id}?type=live&limit=100

# Official leaderboard (from PostgreSQL snapshot)
GET /api/v1/leaderboards/{id}?type=official&limit=100&cursor=0
```

**Response:**
```json
{
  "success": true,
  "data": {
    "leaderboard_id": "global_scores",
    "type": "live",
    "records": [
      {
        "owner_id": "user-123",
        "username": "player1",
        "score": 1000,
        "rank": 1,
        "metadata": {"level": "5"}
      }
    ],
    "next_cursor": "",
    "total_count": 1
  }
}
```

### Get User Rank

```bash
# Live rank
GET /api/v1/leaderboards/{id}/users/{userId}?type=live

# Official rank
GET /api/v1/leaderboards/{id}/users/{userId}?type=official
```

### Get Records Around User

```bash
GET /api/v1/leaderboards/{id}/around/{userId}?type=live&limit=10
```

### Delete User Record

```bash
DELETE /api/v1/leaderboards/{id}/users/{userId}
```

### Create Snapshot (Admin)

```bash
POST /api/v1/snapshots
Content-Type: application/json

{
  "leaderboard_id": "global_scores",
  "max_records": 10000
}
```

### Health Check

```bash
GET /health
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `SERVER_READ_TIMEOUT` | 10s | HTTP read timeout |
| `SERVER_WRITE_TIMEOUT` | 10s | HTTP write timeout |
| `NAKAMA_HOST` | localhost | Nakama server host |
| `NAKAMA_HTTP_PORT` | 7350 | Nakama HTTP API port |
| `NAKAMA_GRPC_PORT` | 7349 | Nakama gRPC port |
| `NAKAMA_SERVER_KEY` | defaultkey | Nakama server key |
| `NAKAMA_USE_SSL` | false | Use SSL for Nakama |
| `POSTGRES_HOST` | localhost | PostgreSQL host |
| `POSTGRES_PORT` | 5432 | PostgreSQL port |
| `POSTGRES_USER` | postgres | PostgreSQL user |
| `POSTGRES_PASSWORD` | postgres | PostgreSQL password |
| `POSTGRES_DB` | leaderboard | PostgreSQL database |
| `POSTGRES_SSLMODE` | disable | PostgreSQL SSL mode |
| `WORKER_ENABLED` | true | Enable snapshot worker |
| `WORKER_SNAPSHOT_INTERVAL` | 30m | Snapshot interval |
| `SNAPSHOT_LEADERBOARDS` | | Comma-separated list of leaderboards to snapshot |

## Setting Up Leaderboards in Nakama

Before using the API, you need to create leaderboards in Nakama. You can do this via:

### Option 1: Nakama Console

1. Open http://localhost:7351
2. Login with admin/password
3. Go to Runtime Modules > Leaderboards
4. Create a new leaderboard

### Option 2: Nakama Server Runtime (Recommended for Production)

Create a `main.go` file in your Nakama server module:

```go
package main

import (
    "context"
    "database/sql"
    "github.com/heroiclabs/nakama-common/runtime"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
    // Create leaderboards on server startup
    err := nk.LeaderboardCreate(ctx, "global_scores", false, "desc", "best", "", nil)
    if err != nil {
        logger.Error("Failed to create leaderboard: %v", err)
    }
    
    err = nk.LeaderboardCreate(ctx, "weekly_scores", false, "desc", "best", "0 0 * * 0", nil)
    if err != nil {
        logger.Error("Failed to create weekly leaderboard: %v", err)
    }
    
    return nil
}
```

### Option 3: HTTP API (Development)

Use the Nakama HTTP API directly:

```bash
curl -X POST http://localhost:7350/v2/console/leaderboard \
  -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "global_scores",
    "sort_order": "desc",
    "operator": "best"
  }'
```

## Project Structure

```
.
├── cmd/
│   └── api/              # Application entrypoint
├── internal/
│   ├── config/           # Configuration loading
│   ├── http/             # HTTP handlers & router
│   ├── leaderboard/      # Core business logic & types
│   ├── nakama/           # Nakama HTTP client
│   ├── postgres/         # PostgreSQL repository
│   └── worker/           # Background snapshot worker
├── migrations/           # SQL migrations
├── docker-compose.yml    # Docker setup
├── Dockerfile           # Multi-stage build
└── README.md
```

## Performance Testing

To compare Nakama vs other solutions (like Redis), you can use tools like:

```bash
# Install hey (HTTP load generator)
go install github.com/rakyll/hey@latest

# Test score submission
hey -n 10000 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -d '{"leaderboard_id":"global_scores","user_id":"test-user","score":1000}' \
  http://localhost:8080/api/v1/scores

# Test leaderboard retrieval
hey -n 10000 -c 100 \
  http://localhost:8080/api/v1/leaderboards/global_scores?type=live&limit=100
```

## Why Nakama?

Nakama provides several advantages for leaderboard systems:

1. **Built-in Leaderboards**: Optimized data structures for ranking
2. **Atomic Operations**: Safe concurrent score updates
3. **Flexible Operators**: Best score, set, increment, decrement
4. **Reset Schedules**: Automatic periodic resets (daily, weekly, etc.)
5. **Subscore Support**: Secondary sorting criteria
6. **Metadata**: Store additional data with each score
7. **Around-User Queries**: Efficient retrieval of nearby rankings

## License

MIT

