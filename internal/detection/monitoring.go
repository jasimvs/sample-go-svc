package detection

import (
	"fmt"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

type Rule interface {
	DetectSuspiciousActivity(txn model.Transaction) (bool, string)
}

type Manager struct {
	transactionChannel <-chan model.Transaction
	rules              []Rule
}

func NewManager(transactionChannel <-chan model.Transaction, rules ...Rule) *Manager {
	return &Manager{
		transactionChannel: transactionChannel,
		rules:              rules,
	}
}

func (m *Manager) RunInBackground() {
	go func() {
		// todo handle clean exit
		for txn := range m.transactionChannel {
			suspicious, flaggedRules := m.DetectSuspiciousActivity(txn)
			// write to DB
			fmt.Println(suspicious, flaggedRules)
		}
	}()
}

func (m *Manager) DetectSuspiciousActivity(txn model.Transaction) (suspicious bool, flaggedRules []string) {
	for _, proc := range m.rules {
		s, f := proc.DetectSuspiciousActivity(txn)
		if s {
			suspicious = true
			flaggedRules = append(flaggedRules, f)
		}
	}
	return suspicious, flaggedRules
}
