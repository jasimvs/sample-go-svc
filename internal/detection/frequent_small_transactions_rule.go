package detection

import (
	"context"
	"time"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

const frequentSmallTransactionsRuleName = "FrequentSmallTransactions"

type FrequentSmallTransactionsRule struct {
	repo            Repository
	MaxCount        int
	ThresholdAmount float64
	WindowDuration  time.Duration
}

func NewFrequentSmallTransactionsRule(repo Repository, maxCount int, thresholdAmount float64, windowDuration time.Duration) *FrequentSmallTransactionsRule {
	if repo == nil {
		panic("Repository cannot be nil for FrequentSmallTransactionsRule")
	}
	return &FrequentSmallTransactionsRule{
		repo:            repo,
		MaxCount:        maxCount,
		ThresholdAmount: thresholdAmount,
		WindowDuration:  windowDuration,
	}
}
func (r *FrequentSmallTransactionsRule) DetectSuspiciousActivity(txn model.Transaction) (bool, string, error) {
	if txn.Amount >= r.ThresholdAmount {
		return false, "", nil
	}

	windowStart := txn.Timestamp.Add(-r.WindowDuration)

	filters := Filter{
		UserID:         txn.UserID,
		AmountLessThan: &r.ThresholdAmount,
		Since:          &windowStart,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recentTxns, err := r.repo.Get(ctx, filters)
	if err != nil {
		return false, "", err
	}

	count := len(recentTxns)
	if count > r.MaxCount {
		return true, frequentSmallTransactionsRuleName, nil
	}

	return false, "", nil
}
