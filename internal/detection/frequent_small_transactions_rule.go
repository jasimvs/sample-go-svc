package detection

import (
	"context"
	"time"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

const frequentSmallTransactionsRuleName = "FrequentSmallTransactions"

type FrequentSmallTransactionsRule struct {
	repo            Repository
	maxCount        int
	thresholdAmount float64
	windowDuration  time.Duration
}

func NewFrequentSmallTransactionsRule(repo Repository, maxCount int, thresholdAmount float64, windowDuration time.Duration) *FrequentSmallTransactionsRule {
	if repo == nil {
		panic("Repository cannot be nil for FrequentSmallTransactionsRule")
	}
	return &FrequentSmallTransactionsRule{
		repo:            repo,
		maxCount:        maxCount,
		thresholdAmount: thresholdAmount,
		windowDuration:  windowDuration,
	}
}
func (r *FrequentSmallTransactionsRule) DetectSuspiciousActivity(txn model.Transaction) (bool, string, error) {
	if txn.Amount >= r.thresholdAmount {
		return false, "", nil
	}

	windowStart := txn.Timestamp.Add(-r.windowDuration)

	filters := Filter{
		UserID:         txn.UserID,
		AmountLessThan: &r.thresholdAmount,
		Since:          &windowStart,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recentTxns, err := r.repo.Get(ctx, filters)
	if err != nil {
		return false, "", err
	}

	count := len(recentTxns)
	if count > r.maxCount {
		return true, frequentSmallTransactionsRuleName, nil
	}

	return false, "", nil
}
