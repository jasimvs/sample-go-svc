package detection

import (
	"github.com/jasimvs/sample-go-svc/internal/model"
)

const highVolumeRuleName = "HighVolumeTransaction"
const amountThreshold = 10000 // todo make it configurable

type HighVolumeRule struct {
	AmountThreshold float64
}

func NewHighVolumeRule() *HighVolumeRule {
	return &HighVolumeRule{
		AmountThreshold: amountThreshold,
	}
}

func (r *HighVolumeRule) DetectSuspiciousActivity(txn model.Transaction) (suspicious bool, flaggedRules string) {
	if txn.Amount > r.AmountThreshold {
		return true, highVolumeRuleName
	}
	return false, ""
}
