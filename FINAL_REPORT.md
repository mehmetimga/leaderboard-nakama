# Leaderboard Service - Final Report

## 🎯 Project Overview

A production-ready leaderboard service built with **Go** and **Nakama**, featuring real-time rankings via WebSocket and a React UI.

## ✅ Test Results

| Test Suite | Status | Count |
|------------|--------|-------|
| Handler Tests | ✅ PASS | 9/9 |
| Service Tests | ✅ PASS | 8/8 |
| **Total** | **✅ PASS** | **17/17** |

## 🏗️ Build Status

```
✅ Build: SUCCESS
✅ Binary: bin/leaderboard-api
```

## 🚀 Quick Start

### 1. Start Services

```bash
# Start Nakama + PostgreSQL
docker-compose up -d postgres nakama

# Wait for Nakama to initialize (creates leaderboards via Lua module)
sleep 15

# Start the API
POSTGRES_PORT=5434 POSTGRES_PASSWORD=localdb POSTGRES_DB=nakama go run ./cmd/api

# In another terminal, start the frontend
cd web && npm run dev
```

### 2. Open the UI

Open http://localhost:5173 in your browser.

### 3. Seed Data & Test

```bash
# Seed 20 random users
./scripts/seed_data.sh

# Start live feed (continuous score updates)
INTERVAL=2 ./scripts/live_feed.sh
```

## 📊 Latest Leaderboard (153 players)

| Rank | Player | Score |
|------|--------|-------|
| 🥇 1 | SwiftLegend144 | 35,173 |
| 🥈 2 | SwiftWarrior448 | 33,222 |
| 🥉 3 | SilentNinja8 | 33,039 |
| 4 | CyberRaven16 | 32,957 |
| 5 | DarkPhantom2 | 32,827 |
| 6 | ThunderPhoenix10 | 32,687 |
| 7 | StormBlaze2 | 32,657 |
| 8 | DarkNinja17 | 32,402 |
| 9 | UltraStorm12 | 31,686 |
| 10 | DarkShadow2 | 31,679 |

## 🌐 Services

| Service | URL | Description |
|---------|-----|-------------|
| Web UI | http://localhost:5173 | React leaderboard dashboard |
| API | http://localhost:8080 | Go REST API |
| WebSocket | ws://localhost:8080/ws | Real-time updates |
| Nakama Console | http://localhost:7351 | admin/password |
| PostgreSQL | localhost:5434 | Database |

## 📡 API Quick Reference

```bash
# Health check
curl http://localhost:8080/health

# Submit scores via Kafka (use demo mode)
make demo  # 10 demo users with real-time rank changes

# Or run Kafka load test
make load-test-kafka  # 1000 messages, ~65K msg/sec

# Get leaderboard (live)
curl "http://localhost:8080/api/v1/leaderboards/global_scores?type=live&limit=10"

# Get user rank
curl "http://localhost:8080/api/v1/leaderboards/global_scores/users/user-123?type=live"
```

## 🧪 Data Seeding Scripts

| Command | Description |
|---------|-------------|
| `make seed` | Seed 20 random users (shell script) |
| `make seed-go` | Seed 50 random users (Go feeder) |
| `make live-feed` | Start continuous score updates |
| `./scripts/seed_data.sh` | Run seed script directly |
| `INTERVAL=5 ./scripts/live_feed.sh` | Custom interval live feed |

## 📁 Project Structure

```
leaderboard-nakama/
├── cmd/api/main.go           # API entrypoint
├── internal/
│   ├── config/               # Configuration
│   ├── http/                 # Handlers & router
│   ├── leaderboard/          # Business logic
│   ├── nakama/               # Nakama client
│   ├── postgres/             # PostgreSQL repository
│   ├── websocket/            # WebSocket hub
│   └── worker/               # Background snapshot worker
├── scripts/
│   ├── seed_data.sh          # Bash seed script
│   ├── live_feed.sh          # Continuous feed script
│   └── feed_data.go          # Go data feeder
├── web/                      # React frontend
├── nakama-module/main.lua    # Nakama runtime (creates leaderboards)
├── migrations/               # SQL migrations
├── docker-compose.yml        # Docker setup
├── Makefile                  # Build commands
└── README.md                 # Documentation
```

## 🔧 Key Features Implemented

- ✅ Submit scores to leaderboards
- ✅ Real-time leaderboard updates via WebSocket
- ✅ Live rankings from Nakama
- ✅ Official/snapshot rankings from PostgreSQL
- ✅ User rank lookup
- ✅ Records around user
- ✅ Background snapshot worker
- ✅ Beautiful React UI with gaming aesthetic
- ✅ Connection status indicators
- ✅ Random data generation
- ✅ Docker Compose setup
- ✅ Comprehensive tests

## 🎮 Real-time Testing Verified

1. ✅ WebSocket connection established
2. ✅ Score submission works via UI and API
3. ✅ Leaderboard displays with 153+ players
4. ✅ Rankings update in real-time
5. ✅ Generate Random button works
6. ✅ Seed scripts successfully populate data

## 📈 Performance Testing

```bash
# Install hey load testing tool
go install github.com/rakyll/hey@latest

# Test score submission (1000 requests, 50 concurrent)
make load-test-write

# Test leaderboard retrieval
make load-test-read
```

---

**Created:** December 4, 2025  
**Stack:** Go 1.23 + Nakama 3.20 + PostgreSQL 16 + React + Vite
