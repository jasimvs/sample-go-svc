package detection

import (
	"context"
	"time"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

const rapidTransfersRuleName = "RapidTransfers"

type RapidTransfersRule struct {
	repo           Repository
	MinConsecutive int
	WindowDuration time.Duration
}

func NewRapidTransfersRule(repo Repository, minConsecutive int, windowDuration time.Duration) *RapidTransfersRule {
	return &RapidTransfersRule{
		repo:           repo,
		MinConsecutive: minConsecutive,
		WindowDuration: windowDuration,
	}
}

func (r *RapidTransfersRule) DetectSuspiciousActivity(txn model.Transaction) (bool, string, error) {

	windowStart := txn.Timestamp.Add(-r.WindowDuration)

	filters := Filter{
		UserID: txn.UserID,
		Since:  &windowStart,
		Type:   model.TransferType,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recentTxns, err := r.repo.Get(ctx, filters)
	if err != nil {
		return false, "", err
	}

	if len(recentTxns) >= r.MinConsecutive {
		return true, rapidTransfersRuleName, nil
	}

	return false, "", nil
}
