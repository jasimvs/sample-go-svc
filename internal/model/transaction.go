package model

import (
	"time"
)

type Transaction struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"userId" db:"user_id"`
	Amount    float64   `json:"amount" db:"amount"`
	Type      string    `json:"type" db:"type"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}
