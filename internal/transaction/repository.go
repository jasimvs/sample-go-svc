package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type Repository interface {
	Migrate(ctx context.Context) error
	Save(ctx context.Context, tx model.Transaction) error
}

type sqliteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) Repository {
	if db == nil {
		panic("database connection (*sql.DB) is required for NewSQLiteRepository")
	}
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Migrate(ctx context.Context) error {
	query := `
    CREATE TABLE IF NOT EXISTS transactions (
        id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
        amount REAL NOT NULL,
        type TEXT NOT NULL,
        timestamp TIMESTAMP NOT NULL,
		is_suspicious INTEGER NOT NULL DEFAULT 0,
		flagged_rules TEXT
    );`

	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_type_timestamp ON transactions(user_id, type, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_suspicious ON transactions(user_id, is_suspicious);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_amount ON transactions(amount);`,
	}
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create transactions table: %w", err)
	}
	for _, indexQuery := range indexQueries {
		_, err := r.db.ExecContext(ctx, indexQuery)
		if err != nil {
			return err
		}
	}

	fmt.Println("Transaction repository migration successful.")
	return nil
}

func (r *sqliteRepository) Save(ctx context.Context, tx model.Transaction) error {
	query := `INSERT INTO transactions (id, user_id, amount, type, timestamp) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		tx.ID,
		tx.UserID,
		tx.Amount,
		tx.Type,
		tx.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction (id: %s): %w", tx.ID, err)
	}
	return nil
}
