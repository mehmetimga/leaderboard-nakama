package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	brokers     = flag.String("brokers", "localhost:9092", "Kafka brokers")
	topic       = flag.String("topic", "leaderboard-scores", "Kafka topic")
	count       = flag.Int("count", 1000, "Number of messages to send")
	concurrency = flag.Int("concurrency", 50, "Number of concurrent workers")
)

type ScoreSubmission struct {
	LeaderboardID string `json:"leaderboard_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Score         int64  `json:"score"`
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🚀 Kafka Load Test")
	fmt.Println("==================")
	fmt.Printf("📡 Brokers: %s\n", *brokers)
	fmt.Printf("📬 Topic: %s\n", *topic)
	fmt.Printf("📊 Messages: %d\n", *count)
	fmt.Printf("⚡ Concurrency: %d\n", *concurrency)
	fmt.Println()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(*brokers),
		Topic:        *topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
	}
	defer writer.Close()

	ctx := context.Background()

	// Create work channel
	work := make(chan int, *count)
	for i := 0; i < *count; i++ {
		work <- i
	}
	close(work)

	var sent int64
	var errors int64
	var wg sync.WaitGroup

	start := time.Now()

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range work {
				sub := ScoreSubmission{
					LeaderboardID: "global_scores",
					UserID:        fmt.Sprintf("load-test-user-%06d", idx),
					Username:      fmt.Sprintf("LoadUser%d", idx),
					Score:         int64(rand.Intn(100000)),
				}

				data, _ := json.Marshal(sub)
				err := writer.WriteMessages(ctx, kafka.Message{
					Key:   []byte(sub.UserID),
					Value: data,
				})

				if err != nil {
					atomic.AddInt64(&errors, 1)
				} else {
					atomic.AddInt64(&sent, 1)
				}

				// Progress every 100 messages
				current := atomic.LoadInt64(&sent)
				if current%100 == 0 {
					fmt.Printf("\r📤 Sent: %d/%d", current, *count)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("\r📤 Sent: %d/%d\n", sent, *count)
	fmt.Println()
	fmt.Println("✅ Load Test Complete")
	fmt.Println("=====================")
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("📊 Messages sent: %d\n", sent)
	fmt.Printf("❌ Errors: %d\n", errors)
	fmt.Printf("🚀 Throughput: %.2f msg/sec\n", float64(sent)/duration.Seconds())
}
