package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	MessageTypeSubscribe     MessageType = "subscribe"
	MessageTypeUnsubscribe   MessageType = "unsubscribe"
	MessageTypeLeaderboard   MessageType = "leaderboard"
	MessageTypeScoreUpdate   MessageType = "score_update"
	MessageTypeError         MessageType = "error"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type          MessageType              `json:"type"`
	LeaderboardID string                   `json:"leaderboard_id,omitempty"`
	Data          interface{}              `json:"data,omitempty"`
	Error         string                   `json:"error,omitempty"`
	Timestamp     int64                    `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// Hub manages WebSocket clients and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[ws] client connected, total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[ws] client disconnected, total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Check if client is subscribed to this leaderboard
				if message.LeaderboardID != "" {
					client.mu.RLock()
					subscribed := client.subscriptions[message.LeaderboardID]
					client.mu.RUnlock()
					if !subscribed {
						continue
					}
				}

				data, err := json.Marshal(message)
				if err != nil {
					continue
				}

				select {
				case client.send <- data:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastLeaderboardUpdate sends a leaderboard update to all subscribed clients
func (h *Hub) BroadcastLeaderboardUpdate(leaderboardID string, result *leaderboard.LeaderboardResult) {
	h.broadcast <- &Message{
		Type:          MessageTypeLeaderboard,
		LeaderboardID: leaderboardID,
		Data:          result,
		Timestamp:     time.Now().UnixMilli(),
	}
}

// BroadcastScoreUpdate sends a score update notification
func (h *Hub) BroadcastScoreUpdate(leaderboardID string, record *leaderboard.LeaderboardRecord) {
	h.broadcast <- &Message{
		Type:          MessageTypeScoreUpdate,
		LeaderboardID: leaderboardID,
		Data:          record,
		Timestamp:     time.Now().UnixMilli(),
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(onMessage func(*Client, *Message)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.SendError("invalid message format")
			continue
		}

		onMessage(c, &msg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Subscribe adds a leaderboard subscription
func (c *Client) Subscribe(leaderboardID string) {
	c.mu.Lock()
	c.subscriptions[leaderboardID] = true
	c.mu.Unlock()
	log.Printf("[ws] client subscribed to leaderboard: %s", leaderboardID)
}

// Unsubscribe removes a leaderboard subscription
func (c *Client) Unsubscribe(leaderboardID string) {
	c.mu.Lock()
	delete(c.subscriptions, leaderboardID)
	c.mu.Unlock()
	log.Printf("[ws] client unsubscribed from leaderboard: %s", leaderboardID)
}

// Send sends a message to the client
func (c *Client) Send(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Buffer full, skip
	}
}

// SendError sends an error message to the client
func (c *Client) SendError(errMsg string) {
	c.Send(&Message{
		Type:      MessageTypeError,
		Error:     errMsg,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendPong sends a pong response
func (c *Client) SendPong() {
	c.Send(&Message{
		Type:      MessageTypePong,
		Timestamp: time.Now().UnixMilli(),
	})
}

