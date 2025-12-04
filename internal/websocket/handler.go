package websocket

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, check against allowed origins
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub     *Hub
	service *leaderboard.Service
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, service *leaderboard.Service) *Handler {
	return &Handler{
		hub:     hub,
		service: service,
	}
}

// ServeWS handles WebSocket connection requests
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := NewClient(h.hub, conn)
	h.hub.register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump(h.handleMessage)
}

// handleMessage processes incoming WebSocket messages
func (h *Handler) handleMessage(client *Client, msg *Message) {
	switch msg.Type {
	case MessageTypeSubscribe:
		if msg.LeaderboardID == "" {
			client.SendError("leaderboard_id is required")
			return
		}
		client.Subscribe(msg.LeaderboardID)
		
		// Send current leaderboard state
		h.sendCurrentLeaderboard(client, msg.LeaderboardID)

	case MessageTypeUnsubscribe:
		if msg.LeaderboardID == "" {
			client.SendError("leaderboard_id is required")
			return
		}
		client.Unsubscribe(msg.LeaderboardID)

	case MessageTypePing:
		client.SendPong()

	default:
		client.SendError("unknown message type")
	}
}

// sendCurrentLeaderboard sends the current leaderboard state to a client
func (h *Handler) sendCurrentLeaderboard(client *Client, leaderboardID string) {
	ctx := context.Background()
	
	result, err := h.service.GetLeaderboard(ctx, leaderboard.GetLeaderboardRequest{
		LeaderboardID: leaderboardID,
		Type:          leaderboard.LiveLeaderboard,
		Limit:         100,
	})
	if err != nil {
		client.SendError("failed to fetch leaderboard: " + err.Error())
		return
	}

	client.Send(&Message{
		Type:          MessageTypeLeaderboard,
		LeaderboardID: leaderboardID,
		Data:          result,
	})
}

// GetHub returns the WebSocket hub
func (h *Handler) GetHub() *Hub {
	return h.hub
}

