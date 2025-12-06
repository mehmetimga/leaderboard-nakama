# Load Test Results

## Kafka Load Test

**Date:** December 6, 2025

### Test Configuration

| Parameter | Value |
|-----------|-------|
| Messages | 1,000 |
| Concurrency | 50 workers |
| Broker | localhost:9092 |
| Topic | leaderboard-scores |

### Results

```
🚀 Kafka Load Test
==================
📡 Brokers: localhost:9092
📬 Topic: leaderboard-scores
📊 Messages: 1000
⚡ Concurrency: 50

✅ Load Test Complete
=====================
⏱️  Duration: 15.4ms
📊 Messages sent: 1000
❌ Errors: 0
🚀 Throughput: 64,953 msg/sec
```

### Summary

| Metric | Value |
|--------|-------|
| Duration | 15.4ms |
| Messages Sent | 1,000 |
| Errors | 0 |
| Throughput | **64,953 msg/sec** |
| Latency (avg) | ~0.015ms per message |

### How to Run

```bash
# Default load test (1000 messages, 50 concurrent)
make load-test-kafka

# Custom parameters
go run ./scripts/kafka/loadtest/main.go --count=5000 --concurrency=100
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--brokers` | localhost:9092 | Kafka broker addresses |
| `--topic` | leaderboard-scores | Kafka topic name |
| `--count` | 1000 | Number of messages to send |
| `--concurrency` | 50 | Number of concurrent workers |

## Leaderboard Read Test

```bash
# Run read load test (requires 'hey' tool)
make load-test-read
```

This tests the HTTP API for reading leaderboard data:
- Endpoint: `GET /api/v1/leaderboards/global_scores?type=live&limit=100`
- 1,000 requests with 50 concurrent connections

## Architecture

```
┌─────────────┐     ┌─────────┐     ┌─────────┐     ┌────────┐
│ Load Test   │────▶│  Kafka  │────▶│   API   │────▶│ Nakama │
│ (Producer)  │     │         │     │(Consumer)│     │        │
└─────────────┘     └─────────┘     └─────────┘     └────────┘
     50 workers        async         real-time       persistent
                    batching        processing       storage
```

## Notes

- Kafka async mode enabled for maximum throughput
- Batch timeout set to 10ms for low latency
- Messages are keyed by user_id for partition consistency
- Zero errors indicates stable Kafka cluster connectivity
