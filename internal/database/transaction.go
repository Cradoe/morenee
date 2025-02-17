package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Transaction struct {
	ID                string         `db:"id"`
	SenderWalletID    string         `db:"sender_wallet_id"`
	RecipientWalletID string         `db:"recipient_wallet_id"`
	ReferenceNumber   string         `db:"reference_number"`
	Amount            float64        `db:"amount"`
	Description       sql.NullString `db:"description"`
	Status            string         `db:"status"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         sql.NullTime   `db:"updated_at"`
}

const (
	// TransactionStatusPending indicates that the transaction has been initiated but not yet completed.
	TransactionStatusPending = "pending"

	// TransactionStatusCompleted indicates that the transaction has been successfully processed and finalized.
	// No further action is required once this status is set.
	TransactionStatusCompleted = "completed"

	// TransactionStatusFailed indicates that the transaction could not be completed successfully due to an error.
	// This may be due to insufficient funds, system errors, or other failure conditions.
	TransactionStatusFailed = "failed"

	// TransactionStatusReversed indicates that a previously completed or failed transaction has been reversed.
	// This status is typically used when funds are returned to the sender or adjustments are made to correct errors.
	TransactionStatusReversed = "reversed"
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

func (db *DB) UpdateTransactionStatus(transaction_id string, status string) (bool, error) {
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

// HasExceededDailyLimit checks whether a user has exceeded their daily debit limit based on their transaction history.
// It sums all transactions initiated by the user for the current day with statuses "completed" or "pending".
// The function then compares the total debit amount with the provided daily limit. If the total amount (including the current transaction) exceeds the limit, it returns true; otherwise, false.
func (db *DB) HasExceededDailyLimit(walletID string, intending_amount float64, dailyLimit float64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var totalDebit float64

	// Query to sum the amount of all "completed" or "pending" transactions for the current day
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE sender_wallet_id = $1 
		AND status IN ($2, $3)
		AND DATE(created_at) = CURRENT_DATE
	`

	err := db.GetContext(ctx, &totalDebit, query, walletID, TransactionStatusCompleted, TransactionStatusPending)
	if err != nil {
		return false, err
	}

	// Check if the total debit (including the new debit attempt) exceeds the daily limit
	if totalDebit+intending_amount > dailyLimit {
		return true, nil
	}

	return false, nil
}
