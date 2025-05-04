package detection

import (
	"github.com/jasimvs/sample-go-svc/internal/model"
)

const highVolumeRuleName = "HighVolumeTransaction"
const amountThreshold = 10000

type HighVolumeRule struct {
	amountThreshold float64
}

func NewHighVolumeRule() *HighVolumeRule {
	return &HighVolumeRule{
		amountThreshold: amountThreshold, // todo make it configurable
	}
}

func (r *HighVolumeRule) DetectSuspiciousActivity(txn model.Transaction) (suspicious bool, flaggedRules string, err error) {
	if txn.Amount > r.amountThreshold {
		return true, highVolumeRuleName, nil
	}
	return false, "", nil
}
