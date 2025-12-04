package leaderboard

import (
	"context"
	"fmt"
)

// NakamaClient defines the interface for Nakama operations
type NakamaClient interface {
	SubmitScore(ctx context.Context, sub ScoreSubmission) (*LeaderboardRecord, error)
	GetLeaderboard(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error)
	GetAroundUser(ctx context.Context, leaderboardID, userID string, limit int) (*LeaderboardResult, error)
	GetUserRecords(ctx context.Context, leaderboardID string, userIDs []string) ([]LeaderboardRecord, error)
	DeleteRecord(ctx context.Context, leaderboardID, userID string) error
	HealthCheck(ctx context.Context) error
}

// Repository defines the interface for PostgreSQL operations
type Repository interface {
	SaveSnapshot(ctx context.Context, leaderboardID string, records []LeaderboardRecord) (*LeaderboardSnapshot, error)
	GetLatestSnapshot(ctx context.Context, leaderboardID string) (*LeaderboardSnapshot, error)
	GetOfficialLeaderboard(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error)
	GetUserRank(ctx context.Context, leaderboardID, userID string) (*LeaderboardRecord, error)
	CleanupOldSnapshots(ctx context.Context, leaderboardID string, keepLast int) error
	HealthCheck(ctx context.Context) error
}

// Service handles leaderboard business logic
type Service struct {
	nakama NakamaClient
	repo   Repository
}

// NewService creates a new leaderboard service
func NewService(nakama NakamaClient, repo Repository) *Service {
	return &Service{
		nakama: nakama,
		repo:   repo,
	}
}

// SubmitScore submits a score to the live leaderboard (Nakama)
func (s *Service) SubmitScore(ctx context.Context, sub ScoreSubmission) (*LeaderboardRecord, error) {
	if sub.LeaderboardID == "" {
		return nil, fmt.Errorf("leaderboard_id is required")
	}
	if sub.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	return s.nakama.SubmitScore(ctx, sub)
}

// GetLeaderboard retrieves a leaderboard based on type (live or official)
func (s *Service) GetLeaderboard(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
	if req.LeaderboardID == "" {
		return nil, fmt.Errorf("leaderboard_id is required")
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	switch req.Type {
	case LiveLeaderboard, "":
		// Default to live leaderboard
		return s.nakama.GetLeaderboard(ctx, req)
	case OfficialLeaderboard:
		return s.repo.GetOfficialLeaderboard(ctx, req)
	default:
		return nil, fmt.Errorf("invalid leaderboard type: %s", req.Type)
	}
}

// GetAroundUser retrieves records around a specific user
func (s *Service) GetAroundUser(ctx context.Context, leaderboardID, userID string, limit int, lbType LeaderboardType) (*LeaderboardResult, error) {
	if leaderboardID == "" {
		return nil, fmt.Errorf("leaderboard_id is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if limit <= 0 {
		limit = 10
	}

	switch lbType {
	case LiveLeaderboard, "":
		return s.nakama.GetAroundUser(ctx, leaderboardID, userID, limit)
	case OfficialLeaderboard:
		// For official, just get the user's record since we don't have around-user in snapshots
		rec, err := s.repo.GetUserRank(ctx, leaderboardID, userID)
		if err != nil {
			return nil, err
		}
		records := []LeaderboardRecord{}
		if rec != nil {
			records = append(records, *rec)
		}
		return &LeaderboardResult{
			LeaderboardID: leaderboardID,
			Type:          OfficialLeaderboard,
			Records:       records,
		}, nil
	default:
		return nil, fmt.Errorf("invalid leaderboard type: %s", lbType)
	}
}

// GetUserRank retrieves a user's rank from either live or official leaderboard
func (s *Service) GetUserRank(ctx context.Context, leaderboardID, userID string, lbType LeaderboardType) (*LeaderboardRecord, error) {
	if leaderboardID == "" {
		return nil, fmt.Errorf("leaderboard_id is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	switch lbType {
	case LiveLeaderboard, "":
		records, err := s.nakama.GetUserRecords(ctx, leaderboardID, []string{userID})
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, nil
		}
		return &records[0], nil
	case OfficialLeaderboard:
		return s.repo.GetUserRank(ctx, leaderboardID, userID)
	default:
		return nil, fmt.Errorf("invalid leaderboard type: %s", lbType)
	}
}

// DeleteRecord removes a user's record from the live leaderboard
func (s *Service) DeleteRecord(ctx context.Context, leaderboardID, userID string) error {
	if leaderboardID == "" {
		return fmt.Errorf("leaderboard_id is required")
	}
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}

	return s.nakama.DeleteRecord(ctx, leaderboardID, userID)
}

// CreateSnapshot creates a snapshot of the live leaderboard for official use
func (s *Service) CreateSnapshot(ctx context.Context, leaderboardID string, maxRecords int) (*LeaderboardSnapshot, error) {
	if leaderboardID == "" {
		return nil, fmt.Errorf("leaderboard_id is required")
	}

	if maxRecords <= 0 {
		maxRecords = 10000
	}

	// Fetch all records from Nakama
	var allRecords []LeaderboardRecord
	cursor := ""
	batchSize := 100

	for {
		result, err := s.nakama.GetLeaderboard(ctx, GetLeaderboardRequest{
			LeaderboardID: leaderboardID,
			Limit:         batchSize,
			Cursor:        cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch leaderboard from nakama: %w", err)
		}

		allRecords = append(allRecords, result.Records...)

		if result.NextCursor == "" || len(allRecords) >= maxRecords {
			break
		}
		cursor = result.NextCursor
	}

	// Trim to maxRecords if needed
	if len(allRecords) > maxRecords {
		allRecords = allRecords[:maxRecords]
	}

	// Save snapshot to PostgreSQL
	snapshot, err := s.repo.SaveSnapshot(ctx, leaderboardID, allRecords)
	if err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	return snapshot, nil
}

// HealthCheck verifies all dependencies are healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	if err := s.nakama.HealthCheck(ctx); err != nil {
		return fmt.Errorf("nakama: %w", err)
	}
	if err := s.repo.HealthCheck(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return nil
}

