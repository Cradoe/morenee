package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Transaction struct {
	ID                int            `db:"id"`
	SenderWalletID    int            `db:"sender_wallet_id"`
	RecipientWalletID int            `db:"recipient_wallet_id"`
	ReferenceNumber   string         `db:"reference_number"`
	Amount            float64        `db:"amount"`
	Description       sql.NullString `db:"description"`
	Status            string         `db:"status"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         sql.NullTime   `db:"updated_at"`
}

// define possible transaction status
const (
	TransactionStatusPending   = "pending"
	TransactionStatusCompleted = "completed"
	TransactionStatusFailed    = "failed"
)

func (db *DB) CreateTransaction(transaction *Transaction, tx *sql.Tx) (*Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans Transaction

	query := `
		INSERT INTO transactions (sender_wallet_id, recipient_wallet_id, amount, reference_number, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, reference_number, amount, sender_wallet_id, recipient_wallet_id, created_at`
	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			transaction.SenderWalletID,
			transaction.RecipientWalletID,
			transaction.Amount,
			transaction.ReferenceNumber,
			transaction.Description,
		).Scan(&trans)
		if err != nil {
			return nil, err
		}
	} else {
		err := db.GetContext(ctx, &trans, query,
			transaction.SenderWalletID,
			transaction.RecipientWalletID,
			transaction.Amount,
			transaction.ReferenceNumber,
			transaction.Description,
		)

		if err != nil {
			return nil, err
		}
	}

	return &trans, nil
}

func (db *DB) UpdateTransactionStatus(transaction_id int, status string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
        UPDATE transactions SET status=$1 WHERE id=$2`

	result, err := db.ExecContext(ctx, query, status, transaction_id)
	if err != nil {
		return false, err
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	if rowsAffected == 0 {
		return false, fmt.Errorf("no rows were updated, transaction ID may not exist")
	}

	return true, nil
}

func (db *DB) FindTransactionByReference(reference_number string) (*Transaction, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans Transaction

	query := `
        SELECT reference_number, status, created_at FROM transactions WHERE reference_number=$1`

	err := db.GetContext(ctx, &trans, query, reference_number)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &trans, true, nil
}

func (db *DB) HasExceededDailyLimit(wallet_id int, amount float64) (bool, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	// defer cancel()

	// var trans Transaction

	// query := `
	//     SELECT reference_number, status, created_at FROM transactions WHERE reference_number=$1`

	// err := db.GetContext(ctx, &trans, query, reference_number)

	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return nil, false, nil
	// 	}
	// 	return nil, false, err
	// }

	return false, nil
}
