package transaction

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jasimvs/sample-go-svc/internal/model"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrConflict   = errors.New("resource conflict")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	if repo == nil {
		panic("Repository cannot be nil for transaction.NewService")
	}
	return Service{repo: repo}
}

func (s *Service) CreateTransaction(ctx context.Context, tx model.Transaction) (model.Transaction, error) {
	tx.ID = "tx_" + uuid.NewString()
	log.Printf("Service: Generated new transaction ID: %s", tx.ID)

	if tx.Type == "" {
		return model.Transaction{}, fmt.Errorf("%w: missing required field: type", ErrValidation)
	}

	if !isValidTransactionType(tx.Type) {
		allowedTypes := fmt.Sprintf("'%s', '%s', '%s'", DepositType, WithdrawalType, TransferType)
		return model.Transaction{}, fmt.Errorf("%w: invalid transaction type '%s', must be one of [%s]", ErrValidation, tx.Type, allowedTypes)
	}

	tx.Timestamp = time.Now().UTC()
	log.Printf("Service: Setting transaction timestamp for ID %s to %s", tx.ID, tx.Timestamp)

	log.Printf("Service: Attempting to save transaction ID %s", tx.ID)
	err := s.repo.Save(ctx, tx)
	if err != nil {
		log.Printf("Service: Error saving transaction ID %s: %v", tx.ID, err)
		return model.Transaction{}, fmt.Errorf("failed to save transaction: %w", err)
	}

	log.Printf("Service: Successfully saved transaction ID %s", tx.ID)
	return tx, nil
}

func isValidTransactionType(txType string) bool {
	switch txType {
	case DepositType, WithdrawalType, TransferType:
		return true
	default:
		return false
	}
}
