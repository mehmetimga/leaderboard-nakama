package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ai-campions/leaderboard-nakama/internal/config"
	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

// Repository handles PostgreSQL operations for official leaderboards
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(ctx context.Context, cfg config.PostgresConfig) (*Repository, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Repository{pool: pool}, nil
}

// Close closes the database connection pool
func (r *Repository) Close() {
	r.pool.Close()
}

// RunMigrations creates the necessary tables
func (r *Repository) RunMigrations(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS leaderboard_snapshots (
			id BIGSERIAL PRIMARY KEY,
			leaderboard_id VARCHAR(128) NOT NULL,
			snapshot_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			total_records BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_leaderboard_time 
		 ON leaderboard_snapshots(leaderboard_id, snapshot_time DESC)`,
		`CREATE TABLE IF NOT EXISTS leaderboard_records (
			id BIGSERIAL PRIMARY KEY,
			snapshot_id BIGINT NOT NULL REFERENCES leaderboard_snapshots(id) ON DELETE CASCADE,
			leaderboard_id VARCHAR(128) NOT NULL,
			owner_id VARCHAR(128) NOT NULL,
			username VARCHAR(128),
			score BIGINT NOT NULL,
			subscore BIGINT NOT NULL DEFAULT 0,
			rank BIGINT NOT NULL,
			metadata JSONB,
			create_time TIMESTAMPTZ,
			update_time TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_records_snapshot ON leaderboard_records(snapshot_id)`,
		`CREATE INDEX IF NOT EXISTS idx_records_leaderboard ON leaderboard_records(leaderboard_id)`,
		`CREATE INDEX IF NOT EXISTS idx_records_owner ON leaderboard_records(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_records_rank ON leaderboard_records(snapshot_id, rank)`,
	}

	for _, migration := range migrations {
		if _, err := r.pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
	}

	return nil
}

// SaveSnapshot stores a leaderboard snapshot
func (r *Repository) SaveSnapshot(ctx context.Context, leaderboardID string, records []leaderboard.LeaderboardRecord) (*leaderboard.LeaderboardSnapshot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	snapshotTime := time.Now().UTC()

	// Create snapshot record
	var snapshotID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO leaderboard_snapshots (leaderboard_id, snapshot_time, total_records)
		 VALUES ($1, $2, $3) RETURNING id`,
		leaderboardID, snapshotTime, len(records),
	).Scan(&snapshotID)
	if err != nil {
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}

	// Batch insert records
	if len(records) > 0 {
		batch := &pgx.Batch{}
		for _, rec := range records {
			metadata, _ := json.Marshal(rec.Metadata)
			batch.Queue(
				`INSERT INTO leaderboard_records 
				 (snapshot_id, leaderboard_id, owner_id, username, score, subscore, rank, metadata, create_time, update_time)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				snapshotID, leaderboardID, rec.OwnerID, rec.Username, rec.Score, rec.Subscore, rec.Rank, metadata, rec.CreateTime, rec.UpdateTime,
			)
		}

		results := tx.SendBatch(ctx, batch)
		for range records {
			if _, err := results.Exec(); err != nil {
				results.Close()
				return nil, fmt.Errorf("insert record: %w", err)
			}
		}
		results.Close()
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &leaderboard.LeaderboardSnapshot{
		ID:            snapshotID,
		LeaderboardID: leaderboardID,
		Records:       records,
		SnapshotTime:  snapshotTime,
		TotalRecords:  int64(len(records)),
	}, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a leaderboard
func (r *Repository) GetLatestSnapshot(ctx context.Context, leaderboardID string) (*leaderboard.LeaderboardSnapshot, error) {
	var snapshot leaderboard.LeaderboardSnapshot
	err := r.pool.QueryRow(ctx,
		`SELECT id, leaderboard_id, snapshot_time, total_records
		 FROM leaderboard_snapshots
		 WHERE leaderboard_id = $1
		 ORDER BY snapshot_time DESC
		 LIMIT 1`,
		leaderboardID,
	).Scan(&snapshot.ID, &snapshot.LeaderboardID, &snapshot.SnapshotTime, &snapshot.TotalRecords)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetOfficialLeaderboard retrieves the official leaderboard from the latest snapshot
func (r *Repository) GetOfficialLeaderboard(ctx context.Context, req leaderboard.GetLeaderboardRequest) (*leaderboard.LeaderboardResult, error) {
	// Get latest snapshot
	snapshot, err := r.GetLatestSnapshot(ctx, req.LeaderboardID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return &leaderboard.LeaderboardResult{
			LeaderboardID: req.LeaderboardID,
			Type:          leaderboard.OfficialLeaderboard,
			Records:       []leaderboard.LeaderboardRecord{},
		}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	offset := 0
	if req.Cursor != "" {
		fmt.Sscanf(req.Cursor, "%d", &offset)
	}

	// Query records
	rows, err := r.pool.Query(ctx,
		`SELECT leaderboard_id, owner_id, username, score, subscore, rank, metadata, create_time, update_time
		 FROM leaderboard_records
		 WHERE snapshot_id = $1
		 ORDER BY rank ASC
		 LIMIT $2 OFFSET $3`,
		snapshot.ID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query records: %w", err)
	}
	defer rows.Close()

	var records []leaderboard.LeaderboardRecord
	for rows.Next() {
		var rec leaderboard.LeaderboardRecord
		var metadata []byte
		var createTime, updateTime *time.Time

		if err := rows.Scan(&rec.LeaderboardID, &rec.OwnerID, &rec.Username, &rec.Score, &rec.Subscore, &rec.Rank, &metadata, &createTime, &updateTime); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}

		if metadata != nil {
			_ = json.Unmarshal(metadata, &rec.Metadata)
		}
		if createTime != nil {
			rec.CreateTime = *createTime
		}
		if updateTime != nil {
			rec.UpdateTime = *updateTime
		}

		records = append(records, rec)
	}

	// Handle owner-specific queries
	var ownerRecords []leaderboard.LeaderboardRecord
	if len(req.OwnerIDs) > 0 {
		ownerRows, err := r.pool.Query(ctx,
			`SELECT leaderboard_id, owner_id, username, score, subscore, rank, metadata, create_time, update_time
			 FROM leaderboard_records
			 WHERE snapshot_id = $1 AND owner_id = ANY($2)
			 ORDER BY rank ASC`,
			snapshot.ID, req.OwnerIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("query owner records: %w", err)
		}
		defer ownerRows.Close()

		for ownerRows.Next() {
			var rec leaderboard.LeaderboardRecord
			var metadata []byte
			var createTime, updateTime *time.Time

			if err := ownerRows.Scan(&rec.LeaderboardID, &rec.OwnerID, &rec.Username, &rec.Score, &rec.Subscore, &rec.Rank, &metadata, &createTime, &updateTime); err != nil {
				return nil, fmt.Errorf("scan owner record: %w", err)
			}

			if metadata != nil {
				_ = json.Unmarshal(metadata, &rec.Metadata)
			}
			if createTime != nil {
				rec.CreateTime = *createTime
			}
			if updateTime != nil {
				rec.UpdateTime = *updateTime
			}

			ownerRecords = append(ownerRecords, rec)
		}
	}

	// Calculate next cursor
	nextCursor := ""
	if len(records) == limit {
		nextCursor = fmt.Sprintf("%d", offset+limit)
	}

	return &leaderboard.LeaderboardResult{
		LeaderboardID: req.LeaderboardID,
		Type:          leaderboard.OfficialLeaderboard,
		Records:       records,
		OwnerRecords:  ownerRecords,
		NextCursor:    nextCursor,
		TotalCount:    snapshot.TotalRecords,
		SnapshotTime:  &snapshot.SnapshotTime,
	}, nil
}

// GetUserRank retrieves a specific user's rank from the official leaderboard
func (r *Repository) GetUserRank(ctx context.Context, leaderboardID, userID string) (*leaderboard.LeaderboardRecord, error) {
	snapshot, err := r.GetLatestSnapshot(ctx, leaderboardID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, nil
	}

	var rec leaderboard.LeaderboardRecord
	var metadata []byte
	var createTime, updateTime *time.Time

	err = r.pool.QueryRow(ctx,
		`SELECT leaderboard_id, owner_id, username, score, subscore, rank, metadata, create_time, update_time
		 FROM leaderboard_records
		 WHERE snapshot_id = $1 AND owner_id = $2`,
		snapshot.ID, userID,
	).Scan(&rec.LeaderboardID, &rec.OwnerID, &rec.Username, &rec.Score, &rec.Subscore, &rec.Rank, &metadata, &createTime, &updateTime)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user rank: %w", err)
	}

	if metadata != nil {
		_ = json.Unmarshal(metadata, &rec.Metadata)
	}
	if createTime != nil {
		rec.CreateTime = *createTime
	}
	if updateTime != nil {
		rec.UpdateTime = *updateTime
	}

	return &rec, nil
}

// CleanupOldSnapshots removes snapshots older than the retention period
func (r *Repository) CleanupOldSnapshots(ctx context.Context, leaderboardID string, keepLast int) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM leaderboard_snapshots
		 WHERE leaderboard_id = $1 AND id NOT IN (
			SELECT id FROM leaderboard_snapshots
			WHERE leaderboard_id = $1
			ORDER BY snapshot_time DESC
			LIMIT $2
		 )`,
		leaderboardID, keepLast,
	)
	return err
}

// HealthCheck verifies database connectivity
func (r *Repository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

