// feed_data.go - Go script for high-performance data feeding
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	apiURL        = flag.String("url", "http://localhost:8080", "API base URL")
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

type ScoreSubmission struct {
	LeaderboardID string `json:"leaderboard_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Score         int64  `json:"score"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

var submitCount int64

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🎮 Leaderboard Data Feeder")
	fmt.Printf("📡 API: %s\n", *apiURL)
	fmt.Printf("🏆 Leaderboard: %s\n", *leaderboardID)

	if *seed {
		fmt.Printf("🌱 Seeding %d users...\n\n", *seedCount)
		seedData()
		return
	}

	fmt.Printf("⏱️  Interval: %v\n", *interval)
	fmt.Printf("💥 Burst: %d per interval\n", *burst)
	fmt.Println("\n🔴 LIVE FEED - Press Ctrl+C to stop\n")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Printf("\n\n📊 Total submissions: %d\n", atomic.LoadInt64(&submitCount))
			fmt.Println("👋 Goodbye!")
			return
		case <-ticker.C:
			for i := 0; i < *burst; i++ {
				go submitRandomScore()
			}
		}
	}
}

func seedData() {
	for i := 0; i < *seedCount; i++ {
		submitRandomScore()
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("\n✅ Seeded %d users!\n", *seedCount)
}

func submitRandomScore() {
	username := generateUsername()
	userID := fmt.Sprintf("user-%s", randomString(8))
	score := rand.Int63n(95000) + 5000

	submission := ScoreSubmission{
		LeaderboardID: *leaderboardID,
		UserID:        userID,
		Username:      username,
		Score:         score,
	}

	body, _ := json.Marshal(submission)
	resp, err := httpClient.Post(
		*apiURL+"/api/v1/scores",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)

	count := atomic.AddInt64(&submitCount, 1)
	timestamp := time.Now().Format("15:04:05")

	if apiResp.Success {
		icon := "📊"
		if score > 80000 {
			icon = "🏆"
		} else if score > 50000 {
			icon = "⭐"
		}
		fmt.Printf("[%s] %s #%d %s: %d pts\n", timestamp, icon, count, username, score)
	} else {
		fmt.Printf("[%s] ❌ #%d Failed: %s\n", timestamp, count, apiResp.Error)
	}
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

