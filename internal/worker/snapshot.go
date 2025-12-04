package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ai-campions/leaderboard-nakama/internal/config"
	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

// SnapshotWorker periodically creates leaderboard snapshots
type SnapshotWorker struct {
	service       *leaderboard.Service
	interval      time.Duration
	leaderboards  []string
	maxRecords    int
	keepSnapshots int
	repo          leaderboard.Repository

	stopCh chan struct{}
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewSnapshotWorker creates a new snapshot worker
func NewSnapshotWorker(
	service *leaderboard.Service,
	repo leaderboard.Repository,
	cfg config.WorkerConfig,
) *SnapshotWorker {
	return &SnapshotWorker{
		service:       service,
		repo:          repo,
		interval:      cfg.SnapshotInterval,
		leaderboards:  []string{}, // Will be configured dynamically
		maxRecords:    10000,
		keepSnapshots: 10, // Keep last 10 snapshots per leaderboard
		stopCh:        make(chan struct{}),
	}
}

// RegisterLeaderboard adds a leaderboard to be snapshotted
func (w *SnapshotWorker) RegisterLeaderboard(leaderboardID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, id := range w.leaderboards {
		if id == leaderboardID {
			return // Already registered
		}
	}
	w.leaderboards = append(w.leaderboards, leaderboardID)
}

// SetMaxRecords sets the maximum number of records per snapshot
func (w *SnapshotWorker) SetMaxRecords(max int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.maxRecords = max
}

// SetKeepSnapshots sets how many snapshots to retain per leaderboard
func (w *SnapshotWorker) SetKeepSnapshots(keep int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.keepSnapshots = keep
}

// Start begins the periodic snapshot worker
func (w *SnapshotWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()
	log.Printf("[worker] snapshot worker started with interval %v", w.interval)
}

// Stop gracefully stops the worker
func (w *SnapshotWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	log.Println("[worker] snapshot worker stopped")
}

func (w *SnapshotWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.runSnapshots(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.runSnapshots(ctx)
		}
	}
}

func (w *SnapshotWorker) runSnapshots(ctx context.Context) {
	w.mu.Lock()
	leaderboards := make([]string, len(w.leaderboards))
	copy(leaderboards, w.leaderboards)
	maxRecords := w.maxRecords
	keepSnapshots := w.keepSnapshots
	w.mu.Unlock()

	if len(leaderboards) == 0 {
		log.Println("[worker] no leaderboards registered for snapshots")
		return
	}

	log.Printf("[worker] starting snapshot run for %d leaderboards", len(leaderboards))

	for _, lbID := range leaderboards {
		if err := w.snapshotLeaderboard(ctx, lbID, maxRecords, keepSnapshots); err != nil {
			log.Printf("[worker] error snapshotting leaderboard %s: %v", lbID, err)
		}
	}

	log.Println("[worker] snapshot run completed")
}

func (w *SnapshotWorker) snapshotLeaderboard(ctx context.Context, leaderboardID string, maxRecords, keepSnapshots int) error {
	start := time.Now()

	snapshot, err := w.service.CreateSnapshot(ctx, leaderboardID, maxRecords)
	if err != nil {
		return err
	}

	log.Printf("[worker] created snapshot for %s: %d records in %v",
		leaderboardID, snapshot.TotalRecords, time.Since(start))

	// Cleanup old snapshots
	if err := w.repo.CleanupOldSnapshots(ctx, leaderboardID, keepSnapshots); err != nil {
		log.Printf("[worker] error cleaning up old snapshots for %s: %v", leaderboardID, err)
	}

	return nil
}

// TriggerSnapshot manually triggers a snapshot for a specific leaderboard
func (w *SnapshotWorker) TriggerSnapshot(ctx context.Context, leaderboardID string) (*leaderboard.LeaderboardSnapshot, error) {
	w.mu.Lock()
	maxRecords := w.maxRecords
	w.mu.Unlock()

	return w.service.CreateSnapshot(ctx, leaderboardID, maxRecords)
}

