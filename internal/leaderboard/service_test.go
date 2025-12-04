package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNakamaClient implements NakamaClient for testing
type MockNakamaClient struct {
	SubmitScoreFunc    func(ctx context.Context, sub ScoreSubmission) (*LeaderboardRecord, error)
	GetLeaderboardFunc func(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error)
	GetAroundUserFunc  func(ctx context.Context, leaderboardID, userID string, limit int) (*LeaderboardResult, error)
	GetUserRecordsFunc func(ctx context.Context, leaderboardID string, userIDs []string) ([]LeaderboardRecord, error)
	DeleteRecordFunc   func(ctx context.Context, leaderboardID, userID string) error
	HealthCheckFunc    func(ctx context.Context) error
}

func (m *MockNakamaClient) SubmitScore(ctx context.Context, sub ScoreSubmission) (*LeaderboardRecord, error) {
	if m.SubmitScoreFunc != nil {
		return m.SubmitScoreFunc(ctx, sub)
	}
	return &LeaderboardRecord{
		LeaderboardID: sub.LeaderboardID,
		OwnerID:       sub.UserID,
		Score:         sub.Score,
		Subscore:      sub.Subscore,
		Rank:          1,
	}, nil
}

func (m *MockNakamaClient) GetLeaderboard(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(ctx, req)
	}
	return &LeaderboardResult{
		LeaderboardID: req.LeaderboardID,
		Type:          LiveLeaderboard,
		Records:       []LeaderboardRecord{},
	}, nil
}

func (m *MockNakamaClient) GetAroundUser(ctx context.Context, leaderboardID, userID string, limit int) (*LeaderboardResult, error) {
	if m.GetAroundUserFunc != nil {
		return m.GetAroundUserFunc(ctx, leaderboardID, userID, limit)
	}
	return &LeaderboardResult{
		LeaderboardID: leaderboardID,
		Type:          LiveLeaderboard,
		Records:       []LeaderboardRecord{},
	}, nil
}

func (m *MockNakamaClient) GetUserRecords(ctx context.Context, leaderboardID string, userIDs []string) ([]LeaderboardRecord, error) {
	if m.GetUserRecordsFunc != nil {
		return m.GetUserRecordsFunc(ctx, leaderboardID, userIDs)
	}
	return []LeaderboardRecord{}, nil
}

func (m *MockNakamaClient) DeleteRecord(ctx context.Context, leaderboardID, userID string) error {
	if m.DeleteRecordFunc != nil {
		return m.DeleteRecordFunc(ctx, leaderboardID, userID)
	}
	return nil
}

func (m *MockNakamaClient) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

// MockRepository implements Repository for testing
type MockRepository struct {
	SaveSnapshotFunc          func(ctx context.Context, leaderboardID string, records []LeaderboardRecord) (*LeaderboardSnapshot, error)
	GetLatestSnapshotFunc     func(ctx context.Context, leaderboardID string) (*LeaderboardSnapshot, error)
	GetOfficialLeaderboardFunc func(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error)
	GetUserRankFunc           func(ctx context.Context, leaderboardID, userID string) (*LeaderboardRecord, error)
	CleanupOldSnapshotsFunc   func(ctx context.Context, leaderboardID string, keepLast int) error
	HealthCheckFunc           func(ctx context.Context) error
}

func (m *MockRepository) SaveSnapshot(ctx context.Context, leaderboardID string, records []LeaderboardRecord) (*LeaderboardSnapshot, error) {
	if m.SaveSnapshotFunc != nil {
		return m.SaveSnapshotFunc(ctx, leaderboardID, records)
	}
	return &LeaderboardSnapshot{
		ID:            1,
		LeaderboardID: leaderboardID,
		Records:       records,
		SnapshotTime:  time.Now(),
		TotalRecords:  int64(len(records)),
	}, nil
}

func (m *MockRepository) GetLatestSnapshot(ctx context.Context, leaderboardID string) (*LeaderboardSnapshot, error) {
	if m.GetLatestSnapshotFunc != nil {
		return m.GetLatestSnapshotFunc(ctx, leaderboardID)
	}
	return nil, nil
}

func (m *MockRepository) GetOfficialLeaderboard(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
	if m.GetOfficialLeaderboardFunc != nil {
		return m.GetOfficialLeaderboardFunc(ctx, req)
	}
	return &LeaderboardResult{
		LeaderboardID: req.LeaderboardID,
		Type:          OfficialLeaderboard,
		Records:       []LeaderboardRecord{},
	}, nil
}

func (m *MockRepository) GetUserRank(ctx context.Context, leaderboardID, userID string) (*LeaderboardRecord, error) {
	if m.GetUserRankFunc != nil {
		return m.GetUserRankFunc(ctx, leaderboardID, userID)
	}
	return nil, nil
}

func (m *MockRepository) CleanupOldSnapshots(ctx context.Context, leaderboardID string, keepLast int) error {
	if m.CleanupOldSnapshotsFunc != nil {
		return m.CleanupOldSnapshotsFunc(ctx, leaderboardID, keepLast)
	}
	return nil
}

func (m *MockRepository) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

func TestService_SubmitScore(t *testing.T) {
	tests := []struct {
		name    string
		sub     ScoreSubmission
		wantErr bool
	}{
		{
			name: "valid submission",
			sub: ScoreSubmission{
				LeaderboardID: "global_scores",
				UserID:        "user-123",
				Score:         1000,
			},
			wantErr: false,
		},
		{
			name: "missing leaderboard_id",
			sub: ScoreSubmission{
				UserID: "user-123",
				Score:  1000,
			},
			wantErr: true,
		},
		{
			name: "missing user_id",
			sub: ScoreSubmission{
				LeaderboardID: "global_scores",
				Score:         1000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&MockNakamaClient{}, &MockRepository{})
			record, err := service.SubmitScore(context.Background(), tt.sub)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, record)
				assert.Equal(t, tt.sub.LeaderboardID, record.LeaderboardID)
				assert.Equal(t, tt.sub.Score, record.Score)
			}
		})
	}
}

func TestService_GetLeaderboard(t *testing.T) {
	ctx := context.Background()

	t.Run("live leaderboard", func(t *testing.T) {
		mockNakama := &MockNakamaClient{
			GetLeaderboardFunc: func(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
				return &LeaderboardResult{
					LeaderboardID: req.LeaderboardID,
					Type:          LiveLeaderboard,
					Records: []LeaderboardRecord{
						{OwnerID: "user-1", Score: 1000, Rank: 1},
						{OwnerID: "user-2", Score: 900, Rank: 2},
					},
				}, nil
			},
		}

		service := NewService(mockNakama, &MockRepository{})
		result, err := service.GetLeaderboard(ctx, GetLeaderboardRequest{
			LeaderboardID: "global_scores",
			Type:          LiveLeaderboard,
			Limit:         10,
		})

		require.NoError(t, err)
		assert.Equal(t, LiveLeaderboard, result.Type)
		assert.Len(t, result.Records, 2)
	})

	t.Run("official leaderboard", func(t *testing.T) {
		snapshotTime := time.Now()
		mockRepo := &MockRepository{
			GetOfficialLeaderboardFunc: func(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
				return &LeaderboardResult{
					LeaderboardID: req.LeaderboardID,
					Type:          OfficialLeaderboard,
					Records: []LeaderboardRecord{
						{OwnerID: "user-1", Score: 950, Rank: 1},
					},
					SnapshotTime: &snapshotTime,
				}, nil
			},
		}

		service := NewService(&MockNakamaClient{}, mockRepo)
		result, err := service.GetLeaderboard(ctx, GetLeaderboardRequest{
			LeaderboardID: "global_scores",
			Type:          OfficialLeaderboard,
			Limit:         10,
		})

		require.NoError(t, err)
		assert.Equal(t, OfficialLeaderboard, result.Type)
		assert.NotNil(t, result.SnapshotTime)
	})

	t.Run("missing leaderboard_id", func(t *testing.T) {
		service := NewService(&MockNakamaClient{}, &MockRepository{})
		_, err := service.GetLeaderboard(ctx, GetLeaderboardRequest{
			Type:  LiveLeaderboard,
			Limit: 10,
		})

		assert.Error(t, err)
	})
}

func TestService_CreateSnapshot(t *testing.T) {
	ctx := context.Background()

	t.Run("successful snapshot", func(t *testing.T) {
		records := []LeaderboardRecord{
			{OwnerID: "user-1", Score: 1000, Rank: 1},
			{OwnerID: "user-2", Score: 900, Rank: 2},
			{OwnerID: "user-3", Score: 800, Rank: 3},
		}

		mockNakama := &MockNakamaClient{
			GetLeaderboardFunc: func(ctx context.Context, req GetLeaderboardRequest) (*LeaderboardResult, error) {
				return &LeaderboardResult{
					LeaderboardID: req.LeaderboardID,
					Type:          LiveLeaderboard,
					Records:       records,
					NextCursor:    "", // No more pages
				}, nil
			},
		}

		mockRepo := &MockRepository{}

		service := NewService(mockNakama, mockRepo)
		snapshot, err := service.CreateSnapshot(ctx, "global_scores", 100)

		require.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, "global_scores", snapshot.LeaderboardID)
		assert.Len(t, snapshot.Records, 3)
	})
}

func TestService_GetUserRank(t *testing.T) {
	ctx := context.Background()

	t.Run("live leaderboard", func(t *testing.T) {
		mockNakama := &MockNakamaClient{
			GetUserRecordsFunc: func(ctx context.Context, leaderboardID string, userIDs []string) ([]LeaderboardRecord, error) {
				return []LeaderboardRecord{
					{LeaderboardID: leaderboardID, OwnerID: userIDs[0], Score: 1000, Rank: 5},
				}, nil
			},
		}

		service := NewService(mockNakama, &MockRepository{})
		record, err := service.GetUserRank(ctx, "global_scores", "user-123", LiveLeaderboard)

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, int64(5), record.Rank)
	})

	t.Run("official leaderboard", func(t *testing.T) {
		mockRepo := &MockRepository{
			GetUserRankFunc: func(ctx context.Context, leaderboardID, userID string) (*LeaderboardRecord, error) {
				return &LeaderboardRecord{
					LeaderboardID: leaderboardID,
					OwnerID:       userID,
					Score:         900,
					Rank:          10,
				}, nil
			},
		}

		service := NewService(&MockNakamaClient{}, mockRepo)
		record, err := service.GetUserRank(ctx, "global_scores", "user-123", OfficialLeaderboard)

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, int64(10), record.Rank)
	})
}

