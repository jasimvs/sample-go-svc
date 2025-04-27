package detection

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupDetectionTestDB creates a test DB, runs migration, and returns the DB and repo.
func setupDetectionTestDB(t *testing.T) (db *sql.DB, repo Repository, cleanup func()) {
	t.Helper()

	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, fmt.Sprintf("test_detection_%s.db", uuid.NewString()[:8]))
	dsn := fmt.Sprintf("%s?_journal=WAL&_busy_timeout=5000&_foreign_keys=on", dbFile)

	var err error
	db, err = sql.Open("sqlite3", dsn)
	require.NoError(t, err, "Failed to open test DB")

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	err = db.Ping()
	require.NoError(t, err, "Failed to ping test DB")

	// Manually run migration (assuming Migrate defines the necessary schema)
	tableQuery := `
    CREATE TABLE IF NOT EXISTS transactions (
        id TEXT PRIMARY KEY, user_id TEXT NOT NULL, amount REAL NOT NULL,
        type TEXT NOT NULL, timestamp TIMESTAMP NOT NULL,
        is_suspicious INTEGER NOT NULL DEFAULT 0, flagged_rules TEXT
    );`
	indexQuery := `
    CREATE INDEX IF NOT EXISTS idx_transactions_is_suspicious ON transactions(is_suspicious);
    `
	_, err = db.Exec(tableQuery)
	require.NoError(t, err)
	_, err = db.Exec(indexQuery)
	require.NoError(t, err)

	// Instantiate the detection repository implementation
	repo, err = NewSQLiteRepository(db) // Use the constructor from this package
	require.NoError(t, err)

	cleanup = func() {
		err := db.Close()
		assert.NoError(t, err, "Failed to close test DB")
	}

	return db, repo, cleanup
}

// Helper to insert test data directly for detection repo tests
func insertTestData(t *testing.T, db *sql.DB, tx Transaction) {
	t.Helper()
	query := `INSERT INTO transactions (id, user_id, amount, type, timestamp, is_suspicious, flagged_rules) VALUES (?, ?, ?, ?, ?, ?, ?)`
	flaggedRulesStr := strings.Join(tx.FlaggedRules, ",")
	_, err := db.Exec(query, tx.ID, tx.UserID, tx.Amount, tx.Type, tx.Timestamp, tx.IsSuspicious, flaggedRulesStr)
	require.NoError(t, err, "Failed to insert test data for tx ID %s", tx.ID)
}

// TestDetectionRepository_Get tests the Get method with various filters.
func TestDetectionRepository_Get(t *testing.T) {
	db, repo, cleanup := setupDetectionTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// --- Setup Data using helper ---
	now := time.Now().UTC().Truncate(time.Second)
	tx1 := Transaction{ID: "det_get_1", UserID: "u1", Amount: 10, Type: "deposit", Timestamp: now.Add(-10 * time.Minute), IsSuspicious: false}
	tx2 := Transaction{ID: "det_get_2", UserID: "u2", Amount: 20, Type: "withdrawal", Timestamp: now.Add(-5 * time.Minute), IsSuspicious: true, FlaggedRules: []string{"RuleC"}}
	tx3 := Transaction{ID: "det_get_3", UserID: "u1", Amount: 150, Type: "transfer", Timestamp: now, IsSuspicious: true, FlaggedRules: []string{"RuleA"}}
	tx4 := Transaction{ID: "det_get_4", UserID: "u1", Amount: 5, Type: "deposit", Timestamp: now.Add(time.Minute), IsSuspicious: false} // Newest
	insertTestData(t, db, tx1)
	insertTestData(t, db, tx2)
	insertTestData(t, db, tx3)
	insertTestData(t, db, tx4)

	// --- Define Bool Pointers ---
	isTrue := true
	isFalse := false

	// --- Test Cases ---
	testCases := []struct {
		name           string
		filters        Filter
		expectedIDs    []string // Expected IDs in DESC timestamp order
		expectedLength int
	}{
		{name: "No Filters", filters: Filter{}, expectedIDs: []string{"det_get_4", "det_get_3", "det_get_2", "det_get_1"}, expectedLength: 4},
		{name: "Filter UserID u1", filters: Filter{UserID: "u1"}, expectedIDs: []string{"det_get_4", "det_get_3", "det_get_1"}, expectedLength: 3},
		{name: "Filter Suspicious True", filters: Filter{IsSuspicious: &isTrue}, expectedIDs: []string{"det_get_3", "det_get_2"}, expectedLength: 2},
		{name: "Filter Suspicious False", filters: Filter{IsSuspicious: &isFalse}, expectedIDs: []string{"det_get_4", "det_get_1"}, expectedLength: 2},
		{name: "Filter Type deposit", filters: Filter{Type: "deposit"}, expectedIDs: []string{"det_get_4", "det_get_1"}, expectedLength: 2},
		{name: "Filter Amount Less Than 25", filters: Filter{AmountLessThan: func(f float64) *float64 { return &f }(25)}, expectedIDs: []string{"det_get_4", "det_get_2", "det_get_1"}, expectedLength: 3},
		{name: "Filter Since 6 minutes ago", filters: Filter{Since: func(t time.Time) *time.Time { return &t }(now.Add(-6 * time.Minute))}, expectedIDs: []string{"det_get_4", "det_get_3", "det_get_2"}, expectedLength: 3},
		{name: "Filter Combined User u1, Type deposit, Since 15 min ago", filters: Filter{UserID: "u1", Type: "deposit", Since: func(t time.Time) *time.Time { return &t }(now.Add(-15 * time.Minute))}, expectedIDs: []string{"det_get_4", "det_get_1"}, expectedLength: 2},
		{name: "Filter Combined User u1, Suspicious true", filters: Filter{UserID: "u1", IsSuspicious: &isTrue}, expectedIDs: []string{"det_get_3"}, expectedLength: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactions, err := repo.Get(ctx, tc.filters)
			require.NoError(t, err, "repo.Get failed")
			require.Len(t, transactions, tc.expectedLength, "Incorrect number of transactions returned")
			retrievedIDs := make([]string, len(transactions))
			for i, tx := range transactions {
				retrievedIDs[i] = tx.ID
			}
			assert.Equal(t, tc.expectedIDs, retrievedIDs, "Transaction IDs mismatch or wrong order")
			// Optional: Add assertions for specific fields if needed
		})
	}
}

// TestDetectionRepository_UpdateSuspicionStatus_Success tests successful update.
func TestDetectionRepository_UpdateSuspicionStatus_Success(t *testing.T) {
	db, repo, cleanup := setupDetectionTestDB(t)
	defer cleanup()
	ctx := context.Background()

	txID := "update_det_1"
	initialTx := Transaction{ID: txID, UserID: "u1", Amount: 100, Type: "deposit", Timestamp: time.Now(), IsSuspicious: false}
	insertTestData(t, db, initialTx)

	updatedRules := []string{"RuleX", "RuleY"}
	err := repo.UpdateSuspicionStatus(ctx, txID, true, updatedRules)
	require.NoError(t, err, "UpdateSuspicionStatus failed")

	// Verify Update using raw DB
	var (
		retrievedIsSuspicious bool
		retrievedFlaggedRules string
	)
	row := db.QueryRowContext(ctx, "SELECT is_suspicious, flagged_rules FROM transactions WHERE id = ?", txID)
	err = row.Scan(&retrievedIsSuspicious, &retrievedFlaggedRules)
	require.NoError(t, err, "Failed to query updated row")
	assert.True(t, retrievedIsSuspicious)
	assert.Equal(t, strings.Join(updatedRules, ","), retrievedFlaggedRules)
}

// TestDetectionRepository_UpdateSuspicionStatus_NotFound tests update on non-existent ID.
func TestDetectionRepository_UpdateSuspicionStatus_NotFound(t *testing.T) {
	_, repo, cleanup := setupDetectionTestDB(t)
	defer cleanup()
	ctx := context.Background()

	err := repo.UpdateSuspicionStatus(ctx, "non_existent_id", true, []string{"RuleZ"})
	require.Error(t, err, "Expected an error when updating non-existent ID")
	// Ensure error is the one defined in the detection package (or imported)
	require.ErrorIs(t, err, ErrUpdateFailed, "Expected specific ErrUpdateFailed")
	assert.Contains(t, err.Error(), "no transaction found with id", "Error message mismatch")
}
