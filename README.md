# Leaderboard Service with Nakama

A production-ready leaderboard service built with Go and [Nakama](https://heroiclabs.com/nakama/), featuring **real-time rankings via WebSocket** and a stunning React UI.

![Architecture](https://img.shields.io/badge/Backend-Go%201.23-00ADD8?style=flat&logo=go)
![Frontend](https://img.shields.io/badge/Frontend-React%20+%20Vite-61DAFB?style=flat&logo=react)
![Nakama](https://img.shields.io/badge/Game%20Backend-Nakama-7B2CBF?style=flat)

## Architecture

```
┌───────────────┐                              ┌─────────────────────────────────┐
│ Game Servers  │                              │      React Web UI (:3000)       │
│               │                              │   Real-time updates via WS      │
└───────┬───────┘                              └───────────────┬─────────────────┘
        │                                                      │
        │ Produce Scores                          HTTP (read)  │  WebSocket (push)
        ▼                                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Kafka (:9092)                                      │
│                        Topic: leaderboard-scores                                │
└───────────────────────────────────┬─────────────────────────────────────────────┘
                                    │
                                    │ Consume Scores
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          Leaderboard API (:8080)                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐          │
│  │    HTTP     │  │  WebSocket   │  │   Kafka     │  │  Background  │          │
│  │  Handlers   │  │     Hub      │  │  Consumer   │  │   Worker     │          │
│  │  (read)     │  │  (push)      │  │  (write)    │  │  (snapshot)  │          │
│  └──────┬──────┘  └──────┬───────┘  └──────┬──────┘  └──────┬───────┘          │
│         │                │                 │                │                  │
│         └────────────────┴────────┬────────┴────────────────┘                  │
│                                   ▼                                             │
│                          ┌──────────────┐                                       │
│                          │  Leaderboard │                                       │
│                          │   Service    │                                       │
│                          └──────┬───────┘                                       │
│                  ┌──────────────┴──────────────┐                                │
│                  ▼                             ▼                                │
│           ┌────────────┐               ┌────────────────┐                       │
│           │   Nakama   │               │   PostgreSQL   │                       │
│           │  (Live LB) │               │ (Official LB)  │                       │
│           └────────────┘               └────────────────┘                       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

| Flow | Path | Description |
|------|------|-------------|
| **Score Ingestion** | Game Server → Kafka → API Consumer → Nakama | All scores via Kafka for scalable ingestion |
| **Read Leaderboard** | Web Client → HTTP GET → API → Nakama | Direct read, no Kafka |
| **Real-time Updates** | API → WebSocket → Web Client | Direct push to clients |

### Key Components

- **Live Leaderboard**: Real-time rankings via Nakama's built-in leaderboard system
- **Official Leaderboard**: Periodic snapshots stored in PostgreSQL (configurable interval, default 30 min)
- **Background Worker**: Automatically snapshots live leaderboards to PostgreSQL
- **Kafka Consumer**: Receives **all** score submissions from Kafka topic
- **WebSocket Hub**: Pushes real-time updates to web clients
- **HTTP Handlers**: Serve leaderboard reads from Nakama/PostgreSQL (read-only)

> **Note**: All score submissions go through Kafka. The HTTP API is read-only for leaderboard queries.
> Web clients receive real-time updates via WebSocket.

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
- **Web UI**: http://localhost:3000 ⭐
- **Leaderboard API**: http://localhost:8080
- **WebSocket**: ws://localhost:8080/ws
- **Nakama Console**: http://localhost:7351 (admin/password)
- **Nakama HTTP API**: http://localhost:7350
- **PostgreSQL**: localhost:5434
- **Kafka**: localhost:9092
- **Zookeeper**: localhost:2181

### Local Development

```bash
# Copy environment file
cp env.example .env

# Start dependencies only
docker-compose up -d postgres nakama

# Run the API locally
go run ./cmd/api

# In another terminal, run the frontend
cd web
npm install
npm run dev
```

### Run Tests

```bash
go test ./...
```

## API Reference

### Submit Score (via Kafka)

Scores are submitted via Kafka for high-throughput ingestion:

```json
// Kafka Topic: leaderboard-scores
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

Use the demo producer to test:
```bash
make demo  # 10 demo users with rank changes
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
| `KAFKA_ENABLED` | false | Enable Kafka consumer |
| `KAFKA_BROKERS` | localhost:9092 | Kafka broker addresses (comma-separated) |
| `KAFKA_TOPIC` | leaderboard-scores | Kafka topic for score messages |
| `KAFKA_GROUP_ID` | leaderboard-consumer | Kafka consumer group ID |

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

## Data Seeding & Live Testing

The project includes scripts for seeding test data and testing real-time functionality.

### Available Scripts

| Script | Description | Usage |
|--------|-------------|-------|
| `scripts/kafka/producer.go` | **Kafka-based data feeder (recommended)** | `go run ./scripts/kafka/producer.go` |
| `scripts/kafka/demo/main.go` | **Demo mode: 10 users with rank changes** | `make demo` |
| `scripts/kafka/loadtest/main.go` | Kafka load test (65K msg/sec) | `make load-test-kafka` |

### Seed Data (via Kafka)

```bash
# Seed 50 users via Kafka
make kafka-seed

# Or run demo mode (10 users with visible rank changes)
make demo
```

**Example Output:**
```
🎮 Demo Leaderboard Producer (10 Users)
=========================================
📡 Brokers: localhost:9092
📬 Topic: leaderboard-scores
🏆 Leaderboard: global_scores

🌱 Seeding 10 demo players...
   #1 🦊 FireFox: 2000000 pts
   #2 🐉 Dragon: 1950000 pts
   #3 🦅 Eagle: 1900000 pts
   ...
✅ Initial seeding complete!
```

### Live Feed (via Kafka)

```bash
# Demo mode: 10 users with visible rank changes (recommended)
make demo

# Fast demo (1 second intervals)
make demo-fast

# Continuous Kafka feed
make kafka-feed
```

**Example Output:**
```
🔄 Starting continuous rank changes...
   Press Ctrl+C to stop

[10:15:24] 🎯 Update #1: 🐺 Wolf
         Score: 1800000 → 1911111 (+111111)
         Rank:  #5 → #3 ⬆️ UP 2
[10:15:26] 🎯 Update #2: 🦂 Scorpion
         Score: 1600000 → 1674438 (+74438)
         Rank:  #9 → #8 ⬆️ UP 1
```

### Kafka Producer

The Kafka producer sends score messages directly to Kafka, which are then consumed by the leaderboard service. This provides better decoupling and scalability.

```bash
# Continuous feed via Kafka (1 submission per second)
go run ./scripts/kafka/producer.go

# Seed mode (50 users via Kafka, then exit)
go run ./scripts/kafka/producer.go --seed --seed-count=50

# Custom interval and burst
go run ./scripts/kafka/producer.go --interval=500ms --burst=3

# Custom Kafka brokers and topic
go run ./scripts/kafka/producer.go --brokers=localhost:9092 --topic=leaderboard-scores
```

**Kafka Producer Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--brokers` | `localhost:9092` | Kafka broker addresses |
| `--topic` | `leaderboard-scores` | Kafka topic |
| `--lb` | `global_scores` | Leaderboard ID |
| `--interval` | `1s` | Time between submissions |
| `--burst` | `1` | Submissions per interval |
| `--seed` | `false` | Seed mode (batch insert, then exit) |
| `--seed-count` | `50` | Number of users to seed |

**Kafka Message Format:**
```json
{
  "leaderboard_id": "global_scores",
  "user_id": "user-abc123",
  "username": "SwiftPhantom42",
  "score": 85432,
  "subscore": 0,
  "metadata": {}
}
```

## Testing Real-time Updates

### Step 1: Start the Stack

```bash
# Terminal 1: Start backend services
docker-compose up -d postgres nakama
sleep 15  # Wait for Nakama to initialize

# Start the API (use port 5434 if local postgres conflicts)
POSTGRES_PORT=5434 POSTGRES_PASSWORD=localdb POSTGRES_DB=nakama go run ./cmd/api
```

```bash
# Terminal 2: Start the frontend
cd web && npm run dev
```

### Step 2: Open the UI

Open http://localhost:5173 in your browser. You should see:
- ✅ **API** indicator (green = connected)
- ✅ **WebSocket** indicator (green = connected)
- 🏆 **Global Rankings** section with player count

### Step 3: Start Demo Mode

```bash
# Terminal 3: Run demo with 10 users
make demo
```

Watch the browser - scores should update in real-time with rank changes!

## Latest Test Results

```
Total Players: 153
┌──────┬──────────────────┬─────────┐
│ Rank │ Player           │ Score   │
├──────┼──────────────────┼─────────┤
│ 🥇 1 │ SwiftLegend144   │ 35,173  │
│ 🥈 2 │ SwiftWarrior448  │ 33,222  │
│ 🥉 3 │ SilentNinja8     │ 33,039  │
│   4  │ CyberRaven16     │ 32,957  │
│   5  │ DarkPhantom2     │ 32,827  │
│   6  │ ThunderPhoenix10 │ 32,687  │
│   7  │ StormBlaze2      │ 32,657  │
│   8  │ DarkNinja17      │ 32,402  │
│   9  │ UltraStorm12     │ 31,686  │
│  10  │ DarkShadow2      │ 31,679  │
└──────┴──────────────────┴─────────┘
```

### Verify via API

```bash
# Get top 10 leaderboard
curl -s "http://localhost:8080/api/v1/leaderboards/global_scores?type=live&limit=10" | jq '.data.records[] | {rank, username, score}'

# Health check with WebSocket client count
curl -s http://localhost:8080/health
# {"success":true,"data":{"status":"healthy","ws_clients":2}}
```

## Performance Testing

To compare Nakama vs other solutions (like Redis), you can use tools like:

```bash
# Test score submission via Kafka (1000 messages, 50 concurrent)
make load-test-kafka

# Test leaderboard retrieval via HTTP
go install github.com/rakyll/hey@latest
hey -n 10000 -c 100 \
  http://localhost:8080/api/v1/leaderboards/global_scores?type=live&limit=100
```

**Kafka Load Test Results:**
```
⏱️  Duration: 15.4ms
📊 Messages sent: 1000
❌ Errors: 0
🚀 Throughput: ~65,000 msg/sec
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

