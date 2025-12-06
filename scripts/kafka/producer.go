// producer.go - Kafka producer for feeding leaderboard scores
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	brokers       = flag.String("brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	topic         = flag.String("topic", "leaderboard-scores", "Kafka topic")
	leaderboardID = flag.String("lb", "global_scores", "Leaderboard ID")
	interval      = flag.Duration("interval", 1*time.Second, "Interval between submissions")
	burst         = flag.Int("burst", 1, "Number of submissions per interval")
	seed          = flag.Bool("seed", false, "Seed mode: submit initial batch and exit")
	seedCount     = flag.Int("seed-count", 50, "Number of users to seed")
)

var adjectives = []string{
	"Dark", "Swift", "Silent", "Fierce", "Mystic", "Cyber", "Neon", "Pixel",
	"Turbo", "Ultra", "Shadow", "Storm", "Thunder", "Blazing", "Frost",
	"Atomic", "Cosmic", "Epic", "Mega", "Super", "Hyper", "Alpha", "Omega",
}

var nouns = []string{
	"Phantom", "Shadow", "Nova", "Blaze", "Storm", "Thunder", "Viper", "Ghost",
	"Raven", "Wolf", "Dragon", "Phoenix", "Titan", "Ninja", "Knight", "Hunter",
	"Warrior", "Legend", "Master", "Champion", "Slayer", "Destroyer", "Avenger",
}

type ScoreMessage struct {
	LeaderboardID string            `json:"leaderboard_id"`
	UserID        string            `json:"user_id"`
	Username      string            `json:"username"`
	Score         int64             `json:"score"`
	Subscore      int64             `json:"subscore,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

var (
	writer      *kafka.Writer
	submitCount int64
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🎮 Kafka Leaderboard Producer")
	fmt.Printf("📡 Brokers: %s\n", *brokers)
	fmt.Printf("📬 Topic: %s\n", *topic)
	fmt.Printf("🏆 Leaderboard: %s\n", *leaderboardID)

	// Initialize Kafka writer
	writer = &kafka.Writer{
		Addr:         kafka.TCP(*brokers),
		Topic:        *topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	defer writer.Close()

	if *seed {
		fmt.Printf("🌱 Seeding %d users...\n\n", *seedCount)
		seedData()
		return
	}

	fmt.Printf("⏱️  Interval: %v\n", *interval)
	fmt.Printf("💥 Burst: %d per interval\n", *burst)
	fmt.Println("\n🔴 LIVE KAFKA FEED - Press Ctrl+C to stop")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	ctx := context.Background()

	for {
		select {
		case <-sigChan:
			fmt.Printf("\n\n📊 Total submissions: %d\n", atomic.LoadInt64(&submitCount))
			fmt.Println("👋 Goodbye!")
			return
		case <-ticker.C:
			for i := 0; i < *burst; i++ {
				go submitRandomScore(ctx)
			}
		}
	}
}

func seedData() {
	ctx := context.Background()
	for i := 0; i < *seedCount; i++ {
		submitRandomScore(ctx)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("\n✅ Seeded %d users via Kafka!\n", *seedCount)
}

func submitRandomScore(ctx context.Context) {
	username := generateUsername()
	userID := fmt.Sprintf("user-%s", randomString(8))
	score := rand.Int63n(95000) + 5000

	msg := ScoreMessage{
		LeaderboardID: *leaderboardID,
		UserID:        userID,
		Username:      username,
		Score:         score,
	}

	body, _ := json.Marshal(msg)

	err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(userID),
		Value: body,
	})

	count := atomic.AddInt64(&submitCount, 1)
	timestamp := time.Now().Format("15:04:05")

	if err != nil {
		fmt.Printf("[%s] ❌ #%d Failed: %v\n", timestamp, count, err)
		return
	}

	icon := "📊"
	if score > 80000 {
		icon = "🏆"
	} else if score > 50000 {
		icon = "⭐"
	}
	fmt.Printf("[%s] %s #%d %s: %d pts (via Kafka)\n", timestamp, icon, count, username, score)
}

func generateUsername() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(999) + 1
	return fmt.Sprintf("%s%s%d", adj, noun, num)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
