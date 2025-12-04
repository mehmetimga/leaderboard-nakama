package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
	"github.com/ai-campions/leaderboard-nakama/internal/websocket"
)

// Handler handles HTTP requests for leaderboard operations
type Handler struct {
	service *leaderboard.Service
	wsHub   *websocket.Hub
}

// NewHandler creates a new HTTP handler
func NewHandler(service *leaderboard.Service, wsHub *websocket.Hub) *Handler {
	return &Handler{
		service: service,
		wsHub:   wsHub,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SubmitScoreRequest represents the request body for score submission
type SubmitScoreRequest struct {
	LeaderboardID string            `json:"leaderboard_id"`
	UserID        string            `json:"user_id"`
	Username      string            `json:"username,omitempty"`
	Score         int64             `json:"score"`
	Subscore      int64             `json:"subscore,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SubmitScore handles POST /api/v1/scores
func (h *Handler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	var req SubmitScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	record, err := h.service.SubmitScore(r.Context(), leaderboard.ScoreSubmission{
		LeaderboardID: req.LeaderboardID,
		UserID:        req.UserID,
		Username:      req.Username,
		Score:         req.Score,
		Subscore:      req.Subscore,
		Metadata:      req.Metadata,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast score update via WebSocket
	if h.wsHub != nil {
		h.wsHub.BroadcastScoreUpdate(req.LeaderboardID, record)
		
		// Also broadcast updated leaderboard
		go func() {
			result, err := h.service.GetLeaderboard(r.Context(), leaderboard.GetLeaderboardRequest{
				LeaderboardID: req.LeaderboardID,
				Type:          leaderboard.LiveLeaderboard,
				Limit:         100,
			})
			if err == nil {
				h.wsHub.BroadcastLeaderboardUpdate(req.LeaderboardID, result)
			}
		}()
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true, Data: record})
}

// GetLeaderboard handles GET /api/v1/leaderboards/{id}
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "id")
	if leaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, "leaderboard_id is required")
		return
	}

	// Parse query parameters
	lbType := leaderboard.LeaderboardType(r.URL.Query().Get("type"))
	if lbType == "" {
		lbType = leaderboard.LiveLeaderboard
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	cursor := r.URL.Query().Get("cursor")

	// Parse owner_ids if provided
	ownerIDs := r.URL.Query()["owner_ids"]

	result, err := h.service.GetLeaderboard(r.Context(), leaderboard.GetLeaderboardRequest{
		LeaderboardID: leaderboardID,
		Type:          lbType,
		Limit:         limit,
		Cursor:        cursor,
		OwnerIDs:      ownerIDs,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true, Data: result})
}

// GetAroundUser handles GET /api/v1/leaderboards/{id}/around/{userId}
func (h *Handler) GetAroundUser(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if leaderboardID == "" || userID == "" {
		h.writeError(w, http.StatusBadRequest, "leaderboard_id and user_id are required")
		return
	}

	lbType := leaderboard.LeaderboardType(r.URL.Query().Get("type"))
	if lbType == "" {
		lbType = leaderboard.LiveLeaderboard
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	result, err := h.service.GetAroundUser(r.Context(), leaderboardID, userID, limit, lbType)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true, Data: result})
}

// GetUserRank handles GET /api/v1/leaderboards/{id}/users/{userId}
func (h *Handler) GetUserRank(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if leaderboardID == "" || userID == "" {
		h.writeError(w, http.StatusBadRequest, "leaderboard_id and user_id are required")
		return
	}

	lbType := leaderboard.LeaderboardType(r.URL.Query().Get("type"))
	if lbType == "" {
		lbType = leaderboard.LiveLeaderboard
	}

	record, err := h.service.GetUserRank(r.Context(), leaderboardID, userID, lbType)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if record == nil {
		h.writeError(w, http.StatusNotFound, "user not found in leaderboard")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true, Data: record})
}

// DeleteScore handles DELETE /api/v1/leaderboards/{id}/users/{userId}
func (h *Handler) DeleteScore(w http.ResponseWriter, r *http.Request) {
	leaderboardID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if leaderboardID == "" || userID == "" {
		h.writeError(w, http.StatusBadRequest, "leaderboard_id and user_id are required")
		return
	}

	if err := h.service.DeleteRecord(r.Context(), leaderboardID, userID); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true})
}

// CreateSnapshotRequest represents the request body for snapshot creation
type CreateSnapshotRequest struct {
	LeaderboardID string `json:"leaderboard_id"`
	MaxRecords    int    `json:"max_records,omitempty"`
}

// CreateSnapshot handles POST /api/v1/snapshots
func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.LeaderboardID == "" {
		h.writeError(w, http.StatusBadRequest, "leaderboard_id is required")
		return
	}

	snapshot, err := h.service.CreateSnapshot(r.Context(), req.LeaderboardID, req.MaxRecords)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Success: true, Data: snapshot})
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.service.HealthCheck(r.Context()); err != nil {
		h.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	wsClients := 0
	if h.wsHub != nil {
		wsClients = h.wsHub.GetClientCount()
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"status":     "healthy",
			"ws_clients": wsClients,
		},
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, Response{Success: false, Error: message})
}
