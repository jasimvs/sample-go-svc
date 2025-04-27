package detection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Filter struct {
	UserID         string
	IsSuspicious   *bool
	Type           string
	AmountLessThan *float64
	Since          *time.Time
}

type Repository interface {
	Get(ctx context.Context, filters Filter) ([]Transaction, error)
	UpdateSuspicionStatus(ctx context.Context, transactionID string, isSuspicious bool, flaggedRules []string) error
}

var (
	ErrUpdateFailed = errors.New("failed to update transaction")
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) (Repository, error) {
	return &sqliteRepository{db: db}, nil
}

// Reusing transactions table, this could be split off into a separate table/DB for scaling
func (r *sqliteRepository) Get(ctx context.Context, filters Filter) ([]Transaction, error) {
	baseQuery := `SELECT id, user_id, amount, type, timestamp, is_suspicious, flagged_rules FROM transactions`
	whereClauses := []string{}
	args := []any{}

	if filters.UserID != "" {
		whereClauses = append(whereClauses, "user_id = ?")
		args = append(args, filters.UserID)
	}
	if filters.IsSuspicious != nil {
		whereClauses = append(whereClauses, "is_suspicious = ?")
		args = append(args, *filters.IsSuspicious)
	}
	if filters.Type != "" {
		whereClauses = append(whereClauses, "type = ?")
		args = append(args, filters.Type)
	}
	if filters.AmountLessThan != nil {
		whereClauses = append(whereClauses, "amount < ?")
		args = append(args, *filters.AmountLessThan)
	}
	if filters.Since != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, *filters.Since)
	}

	query := baseQuery
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY timestamp DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions with filters (%+v): %w", filters, err)
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		var flaggedRulesDB sql.NullString
		err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Type, &tx.Timestamp, &tx.IsSuspicious, &flaggedRulesDB)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		if flaggedRulesDB.Valid && flaggedRulesDB.String != "" {
			tx.FlaggedRules = strings.Split(flaggedRulesDB.String, ",")
		} else {
			tx.FlaggedRules = []string{}
		}
		transactions = append(transactions, tx)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return transactions, nil
}

func (r *sqliteRepository) UpdateSuspicionStatus(ctx context.Context, transactionID string, isSuspicious bool, flaggedRules []string) error {
	query := `UPDATE transactions SET is_suspicious = ?, flagged_rules = ? WHERE id = ?`
	flaggedRulesStr := strings.Join(flaggedRules, ",")

	result, err := r.db.ExecContext(ctx, query, isSuspicious, flaggedRulesStr, transactionID)
	if err != nil {
		return fmt.Errorf("failed to execute update for transaction id %s: %w", transactionID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Warning: Could not get rows affected for update on tx id %s: %v\n", transactionID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: no transaction found with id %s to update", ErrUpdateFailed, transactionID)
	}

	return nil
}
