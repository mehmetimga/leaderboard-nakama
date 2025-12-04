# Role

You are an expert backend engineer specializing in Go, game backends, Nakama, Redis, and PostgreSQL.  
You will create a **leaderboard service** that:

- Uses **Nakama** as the core user + leaderboard backend.
- Uses **PostgreSQL** as the persistent store.
- Optionally uses **Redis** as a cache for some leaderboard queries (if it simplifies performance).
- Is written in **Go**, with clean architecture and tests.

The result should be a small, production-ready style project that I can run locally with Docker and iterate on.

---

# Project Overview

I want a **Leaderboard API** that my other systems can call to:

- **Submit scores** to one or more leaderboards.
- **Fetch live leaderboards** (real-time-ish) using Nakama’s leaderboards.
- **Fetch official / delayed leaderboards** (e.g. 30-minute or periodic snapshots) from PostgreSQL, possibly recomputed by a background job.

There are two conceptual leaderboards:

1. `live` – up-to-date, reading from Nakama’s built-in leaderboard API.
2. `official` – a snapshot stored in PostgreSQL, updated periodically (batch job) and optionally cached in Redis.

The app itself is a **Go HTTP API** that talks to Nakama and Postgres (and Redis if needed).

---

# Tech Stack

- **Language**: Go (latest stable, e.g. 1.22+)
- **Web framework**: Standard library `net/http` or a lightweight router like `chi` (preferred).
- **DB**: PostgreSQL
- **Cache**: Redis (optional but include wiring and one cached read path)
- **Game backend**: Nakama server
- **Container**: Docker + Docker Compose
- **Config**: Environment variables or `.env` (use `github.com/joho/godotenv` if needed)
- **Testing**: Go’s testing package + `testify` (assert/require)

---

# High-Level Architecture

Monorepo-style structure:

```text
.
├── cmd/
│   └── api/            # main.go for HTTP API server
├── internal/
│   ├── config/         # config loading (env, defaults)
│   ├── http/           # HTTP handlers, routing
│   ├── leaderboard/    # core business logic
│   ├── nakama/         # Nakama client wrapper
│   ├── postgres/       # Postgres repository impl
│   ├── redis/          # Redis client wrapper (optional caching)
│   └── worker/         # batch job for official leaderboard snapshots
├── migrations/         # SQL migrations for Postgres
├── docker-compose.yml
├── Dockerfile
└── README.md
