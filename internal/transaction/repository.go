package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

var ErrTransactionNotFound = errors.New("transaction not found")

const (
	DepositType    = "deposit"
	WithdrawalType = "withdrawal"
	TransferType   = "transfer"
)

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
        amount REAL NOT NULL,
        type TEXT NOT NULL,
        timestamp TIMESTAMP NOT NULL
    );`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create transactions table: %w", err)
	}
	fmt.Println("Transaction repository migration successful (type as TEXT).")
	return nil
}

func (r *sqliteRepository) Save(ctx context.Context, tx model.Transaction) error {
	query := `INSERT INTO transactions (id, amount, type, timestamp) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		tx.ID,
		tx.Amount,
		tx.Type,
		tx.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction (id: %s): %w", tx.ID, err)
	}
	return nil
}
