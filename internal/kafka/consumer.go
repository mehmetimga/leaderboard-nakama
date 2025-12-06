package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/ai-campions/leaderboard-nakama/internal/config"
	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
	"github.com/ai-campions/leaderboard-nakama/internal/websocket"
)

// ScoreMessage represents a score submission message from Kafka
type ScoreMessage struct {
	LeaderboardID string            `json:"leaderboard_id"`
	UserID        string            `json:"user_id"`
	Username      string            `json:"username,omitempty"`
	Score         int64             `json:"score"`
	Subscore      int64             `json:"subscore,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Consumer handles Kafka message consumption for leaderboard scores
type Consumer struct {
	reader  *kafka.Reader
	service *leaderboard.Service
	wsHub   *websocket.Hub
	cfg     config.KafkaConfig
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig, service *leaderboard.Service, wsHub *websocket.Hub) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader:  reader,
		service: service,
		wsHub:   wsHub,
		cfg:     cfg,
	}
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		log.Printf("Kafka consumer started - topic: %s, brokers: %v", c.cfg.Topic, c.cfg.Brokers)

		for {
			select {
			case <-ctx.Done():
				log.Println("Kafka consumer stopping...")
				return
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return // Context cancelled, exit gracefully
					}
					log.Printf("Error fetching Kafka message: %v", err)
					continue
				}

				if err := c.processMessage(ctx, msg); err != nil {
					log.Printf("Error processing Kafka message: %v", err)
					// Still commit the message to avoid reprocessing failures
				}

				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Error committing Kafka message: %v", err)
				}
			}
		}
	}()
}

// processMessage handles a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var scoreMsg ScoreMessage
	if err := json.Unmarshal(msg.Value, &scoreMsg); err != nil {
		log.Printf("Failed to unmarshal score message: %v", err)
		return err
	}

	log.Printf("📥 Received score via Kafka: %s - %s: %d pts",
		scoreMsg.LeaderboardID, scoreMsg.Username, scoreMsg.Score)

	// Submit score to Nakama via the service
	record, err := c.service.SubmitScore(ctx, leaderboard.ScoreSubmission{
		LeaderboardID: scoreMsg.LeaderboardID,
		UserID:        scoreMsg.UserID,
		Username:      scoreMsg.Username,
		Score:         scoreMsg.Score,
		Subscore:      scoreMsg.Subscore,
		Metadata:      scoreMsg.Metadata,
	})
	if err != nil {
		log.Printf("Failed to submit score: %v", err)
		return err
	}

	// Broadcast score update via WebSocket
	if c.wsHub != nil {
		c.wsHub.BroadcastScoreUpdate(scoreMsg.LeaderboardID, record)

		// Also broadcast updated leaderboard
		go func() {
			result, err := c.service.GetLeaderboard(ctx, leaderboard.GetLeaderboardRequest{
				LeaderboardID: scoreMsg.LeaderboardID,
				Type:          leaderboard.LiveLeaderboard,
				Limit:         100,
			})
			if err == nil {
				c.wsHub.BroadcastLeaderboardUpdate(scoreMsg.LeaderboardID, result)
			}
		}()
	}

	log.Printf("✅ Score processed: %s (rank: %d)", scoreMsg.Username, record.Rank)
	return nil
}

// Stop gracefully stops the Kafka consumer
func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	if err := c.reader.Close(); err != nil {
		log.Printf("Error closing Kafka reader: %v", err)
	}
	log.Println("Kafka consumer stopped")
}
