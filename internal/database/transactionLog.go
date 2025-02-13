package database

import (
	"context"
	"time"
)

type TransactionLog struct {
	ID            int       `db:"id"`
	UserID        int       `db:"user_id"`
	TransactionID int       `db:"transaction_id"`
	Action        string    `db:"action"`
	CreatedAt     time.Time `db:"created_at"`
}

const (
	TransactionLogActionInitiated = "initiated"
	TransactionLogActionDebit     = "debit"
	TransactionLogActionCredit    = "credit"
	TransactionLogActionFailed    = "failed"
	TransactionLogActionReverted  = "reverted"
	TransactionLogActionSuccess   = "success"
)

func (db *DB) CreateTransactionLog(log *TransactionLog) (*TransactionLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans TransactionLog

	query := `
		INSERT INTO transaction_logs (user_id, transaction_id, action)
		VALUES ($1, $2, $3)
		RETURNING id`

	err := db.GetContext(ctx, &trans, query,
		log.UserID,
		log.TransactionID,
		log.Action,
	)

	if err != nil {
		return &TransactionLog{}, err
	}

	return &trans, nil
}
