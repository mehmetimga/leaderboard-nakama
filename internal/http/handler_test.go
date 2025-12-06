package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

// mockNakamaClient implements leaderboard.NakamaClient for testing
type mockNakamaClient struct {
	getLeaderboardResult *leaderboard.LeaderboardResult
}

func (m *mockNakamaClient) GetLeaderboard(ctx context.Context, req leaderboard.GetLeaderboardRequest) (*leaderboard.LeaderboardResult, error) {
	if m.getLeaderboardResult != nil {
		return m.getLeaderboardResult, nil
	}
	return &leaderboard.LeaderboardResult{
		LeaderboardID: req.LeaderboardID,
		Type:          leaderboard.LiveLeaderboard,
		Records:       []leaderboard.LeaderboardRecord{},
	}, nil
}

func (m *mockNakamaClient) GetAroundUser(ctx context.Context, leaderboardID, userID string, limit int) (*leaderboard.LeaderboardResult, error) {
	return &leaderboard.LeaderboardResult{
		LeaderboardID: leaderboardID,
		Type:          leaderboard.LiveLeaderboard,
		Records:       []leaderboard.LeaderboardRecord{},
	}, nil
}

func (m *mockNakamaClient) GetUserRecords(ctx context.Context, leaderboardID string, userIDs []string) ([]leaderboard.LeaderboardRecord, error) {
	return []leaderboard.LeaderboardRecord{}, nil
}

func (m *mockNakamaClient) DeleteRecord(ctx context.Context, leaderboardID, userID string) error {
	return nil
}

func (m *mockNakamaClient) HealthCheck(ctx context.Context) error {
	return nil
}

// mockRepository implements leaderboard.Repository for testing
type mockRepository struct{}

func (m *mockRepository) SaveSnapshot(ctx context.Context, leaderboardID string, records []leaderboard.LeaderboardRecord) (*leaderboard.LeaderboardSnapshot, error) {
	return &leaderboard.LeaderboardSnapshot{ID: 1, LeaderboardID: leaderboardID}, nil
}

func (m *mockRepository) GetLatestSnapshot(ctx context.Context, leaderboardID string) (*leaderboard.LeaderboardSnapshot, error) {
	return nil, nil
}

func (m *mockRepository) GetOfficialLeaderboard(ctx context.Context, req leaderboard.GetLeaderboardRequest) (*leaderboard.LeaderboardResult, error) {
	return &leaderboard.LeaderboardResult{LeaderboardID: req.LeaderboardID, Type: leaderboard.OfficialLeaderboard}, nil
}

func (m *mockRepository) GetUserRank(ctx context.Context, leaderboardID, userID string) (*leaderboard.LeaderboardRecord, error) {
	return nil, nil
}

func (m *mockRepository) CleanupOldSnapshots(ctx context.Context, leaderboardID string, keepLast int) error {
	return nil
}

func (m *mockRepository) HealthCheck(ctx context.Context) error {
	return nil
}

func TestGetLeaderboardHandler(t *testing.T) {
	t.Run("missing leaderboard id", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboards/", nil)
		rec := httptest.NewRecorder()

		// Create chi context with empty id
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetLeaderboard(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("valid request with default type", func(t *testing.T) {
		mockNakama := &mockNakamaClient{
			getLeaderboardResult: &leaderboard.LeaderboardResult{
				LeaderboardID: "global_scores",
				Type:          leaderboard.LiveLeaderboard,
				Records: []leaderboard.LeaderboardRecord{
					{OwnerID: "user-1", Score: 1000, Rank: 1},
				},
			},
		}
		service := leaderboard.NewService(mockNakama, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboards/global_scores?limit=10", nil)
		rec := httptest.NewRecorder()

		// Set up chi router context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "global_scores")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetLeaderboard(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("official leaderboard type", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboards/global_scores?type=official&limit=10", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "global_scores")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetLeaderboard(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestGetUserRankHandler(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboards/global_scores/users/unknown-user", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "global_scores")
		rctx.URLParams.Add("userId", "unknown-user")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetUserRank(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHealthCheckHandler(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.HealthCheck(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestDeleteScoreHandler(t *testing.T) {
	t.Run("missing parameters", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/leaderboards//users/", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "")
		rctx.URLParams.Add("userId", "")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.DeleteScore(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("successful delete", func(t *testing.T) {
		service := leaderboard.NewService(&mockNakamaClient{}, &mockRepository{})
		handler := NewHandler(service, nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/leaderboards/global_scores/users/user-123", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "global_scores")
		rctx.URLParams.Add("userId", "user-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.DeleteScore(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
