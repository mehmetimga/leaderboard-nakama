-- 001_create_leaderboard_tables.sql
-- Creates tables for official leaderboard snapshots

-- Snapshots table: stores metadata about each snapshot
CREATE TABLE IF NOT EXISTS leaderboard_snapshots (
    id BIGSERIAL PRIMARY KEY,
    leaderboard_id VARCHAR(128) NOT NULL,
    snapshot_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_records BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient lookup by leaderboard and time
CREATE INDEX IF NOT EXISTS idx_snapshots_leaderboard_time 
ON leaderboard_snapshots(leaderboard_id, snapshot_time DESC);

-- Records table: stores individual leaderboard entries
CREATE TABLE IF NOT EXISTS leaderboard_records (
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
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_records_snapshot ON leaderboard_records(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_records_leaderboard ON leaderboard_records(leaderboard_id);
CREATE INDEX IF NOT EXISTS idx_records_owner ON leaderboard_records(owner_id);
CREATE INDEX IF NOT EXISTS idx_records_rank ON leaderboard_records(snapshot_id, rank);

-- Comments
COMMENT ON TABLE leaderboard_snapshots IS 'Stores point-in-time snapshots of Nakama leaderboards';
COMMENT ON TABLE leaderboard_records IS 'Stores individual records from leaderboard snapshots';
COMMENT ON COLUMN leaderboard_records.metadata IS 'JSON metadata associated with the score';

