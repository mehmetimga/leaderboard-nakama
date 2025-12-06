// Demo producer for visible rank changes with only 10 users
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"
)

type ScoreMessage struct {
	LeaderboardID string `json:"leaderboard_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Score         int64  `json:"score"`
}

type Player struct {
	ID       string
	Username string
	Score    int64
	Rank     int
}

var (
	brokers       = flag.String("brokers", "localhost:9092", "Kafka broker addresses")
	topic         = flag.String("topic", "leaderboard-scores", "Kafka topic name")
	leaderboardID = flag.String("leaderboard", "global_scores", "Leaderboard ID")
	interval      = flag.Duration("interval", 2*time.Second, "Interval between score updates")
	resetFirst    = flag.Bool("reset", true, "Reset leaderboard with fresh 10 users first")
)

// 10 demo players with memorable names
var demoPlayers = []struct {
	ID       string
	Username string
	Emoji    string
}{
	{"demo-001", "🦊 FireFox", "🦊"},
	{"demo-002", "🐉 Dragon", "🐉"},
	{"demo-003", "🦅 Eagle", "🦅"},
	{"demo-004", "🦁 Lion", "🦁"},
	{"demo-005", "🐺 Wolf", "🐺"},
	{"demo-006", "🦈 Shark", "🦈"},
	{"demo-007", "🦅 Falcon", "🦅"},
	{"demo-008", "🐻 Bear", "🐻"},
	{"demo-009", "🦂 Scorpion", "🦂"},
	{"demo-010", "🦎 Gecko", "🦎"},
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🎮 Demo Leaderboard Producer (10 Users)")
	fmt.Println("=========================================")
	fmt.Printf("📡 Brokers: %s\n", *brokers)
	fmt.Printf("📬 Topic: %s\n", *topic)
	fmt.Printf("🏆 Leaderboard: %s\n", *leaderboardID)
	fmt.Printf("⏱️  Interval: %s\n", *interval)
	fmt.Println()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(*brokers),
		Topic:        *topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	defer writer.Close()

	ctx := context.Background()

	// Initialize players with spread-out scores
	players := make([]*Player, len(demoPlayers))
	baseScore := int64(10000)
	for i, p := range demoPlayers {
		players[i] = &Player{
			ID:       p.ID,
			Username: p.Username,
			Score:    baseScore - int64(i*1000), // 10000, 9000, 8000, ...
			Rank:     i + 1,
		}
	}

	// Seed initial scores
	if *resetFirst {
		fmt.Println("🌱 Seeding 10 demo players...")
		fmt.Println()
		for _, player := range players {
			if err := sendScore(ctx, writer, player); err != nil {
				fmt.Printf("❌ Failed to seed %s: %v\n", player.Username, err)
			} else {
				fmt.Printf("   #%d %s: %d pts\n", player.Rank, player.Username, player.Score)
			}
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Println()
		fmt.Println("✅ Initial seeding complete!")
		fmt.Println()
		fmt.Println("⏳ Starting rank shuffling in 3 seconds...")
		time.Sleep(3 * time.Second)
	}

	fmt.Println()
	fmt.Println("🔄 Starting continuous rank changes...")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	updateCount := 0
	for range ticker.C {
		updateCount++

		// Pick a random player (not the top player, so there's room to go up)
		playerIdx := rand.Intn(len(players)-1) + 1 // Skip index 0 (top player)
		player := players[playerIdx]

		// Calculate a score that will move them up by 1-3 ranks
		rankJump := rand.Intn(3) + 1 // Jump 1-3 positions
		targetRank := player.Rank - rankJump
		if targetRank < 1 {
			targetRank = 1
		}

		// Find the player at target rank and beat their score
		var targetScore int64
		for _, p := range players {
			if p.Rank == targetRank {
				targetScore = p.Score + int64(rand.Intn(500)+100) // Beat by 100-600
				break
			}
		}

		oldRank := player.Rank
		oldScore := player.Score
		player.Score = targetScore

		// Send the update
		if err := sendScore(ctx, writer, player); err != nil {
			fmt.Printf("❌ Update failed: %v\n", err)
			continue
		}

		// Recalculate all ranks
		recalculateRanks(players)

		// Show the update
		fmt.Printf("[%s] 🎯 Update #%d: %s\n",
			time.Now().Format("15:04:05"),
			updateCount,
			player.Username)
		fmt.Printf("         Score: %d → %d (+%d)\n", oldScore, player.Score, player.Score-oldScore)
		fmt.Printf("         Rank:  #%d → #%d", oldRank, player.Rank)
		if player.Rank < oldRank {
			fmt.Printf(" ⬆️ UP %d", oldRank-player.Rank)
		}
		fmt.Println()

		// Show current standings every 5 updates
		if updateCount%5 == 0 {
			fmt.Println()
			fmt.Println("📊 Current Standings:")
			for _, p := range players {
				medal := "  "
				if p.Rank == 1 {
					medal = "🥇"
				} else if p.Rank == 2 {
					medal = "🥈"
				} else if p.Rank == 3 {
					medal = "🥉"
				}
				fmt.Printf("   %s #%d %s: %d pts\n", medal, p.Rank, p.Username, p.Score)
			}
			fmt.Println()
		}
	}
}

func sendScore(ctx context.Context, writer *kafka.Writer, player *Player) error {
	msg := ScoreMessage{
		LeaderboardID: *leaderboardID,
		UserID:        player.ID,
		Username:      player.Username,
		Score:         player.Score,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(player.ID),
		Value: data,
	})
}

func recalculateRanks(players []*Player) {
	// Sort by score descending
	for i := 0; i < len(players)-1; i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}
	// Assign ranks
	for i, p := range players {
		p.Rank = i + 1
	}
}
