package vimkunja

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

// DescriptionConflictError retains the opened baseline and latest source for
// explicit reload or return-to-draft. Saving never merges these descriptions.
type DescriptionConflictError struct{ Baseline, Current string }

func (e *DescriptionConflictError) Error() string {
	return "Description changed externally; reload or return to draft"
}

func prepareDescriptionDraft(draft TaskDraft) (TaskDraft, error) {
	if c := draft.DescriptionCandidate; c != nil {
		if !c.Valid() {
			return TaskDraft{}, fmt.Errorf("invalid Description save candidate")
		}
		baseline := c.Baseline()
		draft.Description = c.HTML()
		// Continued draft editing may still validate against its original source
		// document after our own successful save. An explicit persistence baseline
		// supersedes that provenance; external changes are still checked in the Tx.
		if draft.DescriptionBaseline == nil {
			draft.DescriptionBaseline = &baseline
		}
	}
	return draft, nil
}

// Pin the connection until commit/rollback cleanup finishes. database/sql marks
// a Tx done even when SQLite COMMIT fails (e.g. SQLITE_BUSY), and modernc leaves
// that underlying transaction open; Tx.Rollback can no longer clean it up.
func beginTaskSave(ctx context.Context, db *sql.DB) (*sql.Tx, *sql.Conn, error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		connection.Close()
		return nil, nil, err
	}
	return tx, connection, nil
}
func commitTaskSave(tx *sql.Tx, connection *sql.Conn) error {
	err := tx.Commit()
	if err == nil {
		return nil
	}
	// Cleanup must survive an expired save context. If rollback fails, discard
	// the connection rather than return a possibly open transaction to the pool.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, cleanupErr := connection.ExecContext(cleanupCtx, "ROLLBACK"); cleanupErr != nil {
		_ = connection.Raw(func(any) error { return driver.ErrBadConn })
	}
	return err
}
