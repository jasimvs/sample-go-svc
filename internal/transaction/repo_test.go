package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jasimvs/sample-go-svc/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite DB or a temporary file DB for testing.
func setupTestDB(t *testing.T) (db *sql.DB, repo Repository, cleanup func()) {
	t.Helper()

	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, fmt.Sprintf("test_transactions_%s.db", uuid.NewString()[:8]))
	dsn := fmt.Sprintf("%s?_journal=WAL&_busy_timeout=5000&_foreign_keys=on", dbFile)

	var err error
	db, err = sql.Open("sqlite3", dsn)
	require.NoError(t, err, "Failed to open test DB")

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	err = db.Ping()
	require.NoError(t, err, "Failed to ping test DB")

	repo = NewSQLiteRepository(db)

	cleanup = func() {
		err := db.Close()
		assert.NoError(t, err, "Failed to close test DB")
	}

	return db, repo, cleanup
}

// TestMigrateSuccess tests that Migrate runs without error and creates the table idempotently.
func TestSQLiteRepository_Migrate(t *testing.T) {
	db, repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// --- First Migration ---
	err := repo.Migrate(ctx)
	require.NoError(t, err, "First migration failed")

	// --- Verify Table Exists (by trying to insert) ---
	_, err = db.ExecContext(ctx, `INSERT INTO transactions (id, user_id, amount, type, timestamp) VALUES (?, ?, ?, ?, ?)`,
		"migrate_test_id", "user_id_1", 1.0, model.DepositType, time.Now())
	require.NoError(t, err, "Failed to insert into table after first migration, table might not exist or schema is wrong")

	// --- Second Migration (Idempotency check) ---
	err = repo.Migrate(ctx)
	require.NoError(t, err, "Second migration (idempotency check) failed")

	_, err = db.ExecContext(ctx, `INSERT INTO transactions (id, user_id, amount, type, timestamp) VALUES (?, ?, ?, ?, ?)`,
		"migrate_test_id_2", "user_id_1", 2.0, model.WithdrawalType, time.Now())
	require.NoError(t, err, "Failed to insert into table after second migration")
}

// TestSaveSuccess tests saving a valid transaction.
func TestSQLiteRepository_Save_Success(t *testing.T) {
	db, repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Migrate first, as Save depends on the table existing
	err := repo.Migrate(ctx)
	require.NoError(t, err, "Migration failed before saving")

	// --- Prepare Test Data ---
	saveTx := model.Transaction{
		ID:        "save_test_" + uuid.NewString()[:8],
		UserID:    "user_id_1",
		Amount:    123.45,
		Type:      model.DepositType,
		Timestamp: time.Now().UTC().Truncate(time.Second), // Truncate for comparison
	}

	// --- Call Save ---
	err = repo.Save(ctx, saveTx)
	require.NoError(t, err, "repo.Save failed")

	// --- Verify Insertion (using raw db connection) ---
	var (
		retrievedID     string
		retrievedUserID string
		retrievedAmount float64
		retrievedType   string
		retrievedTS     time.Time
	)
	query := "SELECT id, user_id, amount, type, timestamp FROM transactions WHERE id = ?"
	row := db.QueryRowContext(ctx, query, saveTx.ID)
	err = row.Scan(&retrievedID, &retrievedUserID, &retrievedAmount, &retrievedType, &retrievedTS)
	require.NoError(t, err, "Failed to query and scan the saved row")

	// --- Assertions ---
	assert.Equal(t, saveTx.ID, retrievedID)
	assert.Equal(t, saveTx.UserID, retrievedUserID)
	assert.Equal(t, saveTx.Amount, retrievedAmount)
	assert.Equal(t, saveTx.Type, retrievedType)

	// Use WithinDuration for time comparison due to potential db precision differences
	assert.WithinDuration(t, saveTx.Timestamp, retrievedTS, time.Second)
}

// TestSaveDuplicateID tests saving a transaction with an existing ID.
func TestSQLiteRepository_Save_DuplicateID(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Migrate first
	err := repo.Migrate(ctx)
	require.NoError(t, err, "Migration failed before saving")

	// --- Prepare Test Data ---
	commonID := "duplicate_save_" + uuid.NewString()[:8]
	tx1 := model.Transaction{
		ID:        commonID,
		UserID:    "user_id_1",
		Amount:    10.0,
		Type:      model.TransferType,
		Timestamp: time.Now(),
	}
	tx2 := model.Transaction{ // Same ID
		ID:        commonID,
		UserID:    "user_id_2",
		Amount:    20.0,
		Type:      model.DepositType,
		Timestamp: time.Now(),
	}

	// --- Save First Tx ---
	err = repo.Save(ctx, tx1)
	require.NoError(t, err, "Saving the first transaction failed")

	// --- Save Second Tx (Should Fail) ---
	err = repo.Save(ctx, tx2)
	require.Error(t, err, "Expected an error when saving with a duplicate ID")
}
