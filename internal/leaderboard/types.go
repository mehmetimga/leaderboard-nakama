package leaderboard

import "time"

// LeaderboardType represents the type of leaderboard
type LeaderboardType string

const (
	// LiveLeaderboard is the real-time leaderboard from Nakama
	LiveLeaderboard LeaderboardType = "live"
	// OfficialLeaderboard is the periodic snapshot stored in PostgreSQL
	OfficialLeaderboard LeaderboardType = "official"
)

// SortOrder defines how scores are sorted
type SortOrder string

const (
	SortOrderDescending SortOrder = "desc" // Higher is better (default)
	SortOrderAscending  SortOrder = "asc"  // Lower is better (e.g., time-based)
)

// Operator defines how scores are handled on submission
type Operator string

const (
	OperatorBest       Operator = "best"       // Keep best score
	OperatorSet        Operator = "set"        // Always overwrite
	OperatorIncrement  Operator = "increment"  // Add to existing
	OperatorDecrement  Operator = "decrement"  // Subtract from existing
)

// LeaderboardConfig defines a leaderboard's settings
type LeaderboardConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SortOrder   SortOrder `json:"sort_order"`
	Operator    Operator  `json:"operator"`
	ResetSchedule string  `json:"reset_schedule,omitempty"` // Cron expression for resets
}

// ScoreSubmission represents a score to be submitted
type ScoreSubmission struct {
	LeaderboardID string            `json:"leaderboard_id"`
	UserID        string            `json:"user_id"`
	Username      string            `json:"username,omitempty"`
	Score         int64             `json:"score"`
	Subscore      int64             `json:"subscore,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// LeaderboardRecord represents a single entry in the leaderboard
type LeaderboardRecord struct {
	LeaderboardID string            `json:"leaderboard_id"`
	OwnerID       string            `json:"owner_id"`
	Username      string            `json:"username,omitempty"`
	Score         int64             `json:"score"`
	Subscore      int64             `json:"subscore,omitempty"`
	Rank          int64             `json:"rank"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreateTime    time.Time         `json:"create_time"`
	UpdateTime    time.Time         `json:"update_time"`
}

// LeaderboardResult represents a paginated leaderboard response
type LeaderboardResult struct {
	LeaderboardID string              `json:"leaderboard_id"`
	Type          LeaderboardType     `json:"type"`
	Records       []LeaderboardRecord `json:"records"`
	OwnerRecords  []LeaderboardRecord `json:"owner_records,omitempty"`
	NextCursor    string              `json:"next_cursor,omitempty"`
	PrevCursor    string              `json:"prev_cursor,omitempty"`
	TotalCount    int64               `json:"total_count,omitempty"`
	SnapshotTime  *time.Time          `json:"snapshot_time,omitempty"` // Only for official leaderboards
}

// GetLeaderboardRequest contains parameters for fetching a leaderboard
type GetLeaderboardRequest struct {
	LeaderboardID string
	Type          LeaderboardType
	Limit         int
	Cursor        string
	OwnerIDs      []string // Optional: get specific users' records
}

// LeaderboardSnapshot represents a point-in-time snapshot for official leaderboards
type LeaderboardSnapshot struct {
	ID            int64               `json:"id"`
	LeaderboardID string              `json:"leaderboard_id"`
	Records       []LeaderboardRecord `json:"records"`
	SnapshotTime  time.Time           `json:"snapshot_time"`
	TotalRecords  int64               `json:"total_records"`
}

