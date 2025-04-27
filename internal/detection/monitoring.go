package detection

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jasimvs/sample-go-svc/internal/model"
)

type Transaction struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	Amount       float64   `json:"amount" db:"amount"`
	Type         string    `json:"type" db:"type"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	IsSuspicious bool      `json:"is_suspicious" db:"is_suspicious"`
	FlaggedRules []string  `json:"flagged_rules" db:"flagged_rules"`
}

type Rule interface {
	DetectSuspiciousActivity(txn model.Transaction) (bool, string, error)
}

type DetectionRepository interface {
	Get(ctx context.Context, filters Filter) ([]Transaction, error)
	UpdateSuspicionStatus(ctx context.Context, transactionID string, isSuspicious bool, flaggedRules []string) error
}

type Manager struct {
	transactionChannel <-chan model.Transaction
	rules              []Rule
	repo               Repository
}

func NewManager(transactionChannel <-chan model.Transaction, repo Repository, rules ...Rule) *Manager {
	return &Manager{
		transactionChannel: transactionChannel,
		rules:              rules,
		repo:               repo,
	}
}

func (m *Manager) RunInBackground() {
	go func() {
		// todo handle clean exit
		for txn := range m.transactionChannel {
			suspicious, flaggedRules, err := m.DetectSuspiciousActivity(txn)
			if err != nil {
				log.Printf("Detection Manager: Error detecting suspicious activity for Tx ID %s: %v", txn.ID, err)
				// todo send to retry queue or dead letter queue
				continue
			}

			if suspicious {
				log.Printf("Detection Manager: Updating suspicion status for Tx ID %s (Suspicious: %t, Rules: %v)", txn.ID, suspicious, flaggedRules)
				err := m.repo.UpdateSuspicionStatus(context.Background(), txn.ID, suspicious, flaggedRules)
				if err != nil {
					log.Printf("failed to update suspicion status for Tx ID %s: %v", txn.ID, err)
				}
			}
			fmt.Println(suspicious, flaggedRules)
		}
	}()
}

func (m *Manager) DetectSuspiciousActivity(txn model.Transaction) (suspicious bool, flaggedRules []string, err error) {
	for _, proc := range m.rules {
		s, f, err := proc.DetectSuspiciousActivity(txn)
		if err != nil {
			return false, nil, err
		}
		if s {
			suspicious = true
			flaggedRules = append(flaggedRules, f)
		}
	}
	return suspicious, flaggedRules, nil
}
