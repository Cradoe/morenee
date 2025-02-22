package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// type Transaction struct {
// 	ID                string         `db:"id"`
// 	SenderWalletID    string         `db:"sender_wallet_id"`
// 	RecipientWalletID string         `db:"recipient_wallet_id"`
// 	ReferenceNumber   string         `db:"reference_number"`
// 	Amount            float64        `db:"amount"`
// 	Description       sql.NullString `db:"description"`
// 	Status            string         `db:"status"`
// 	CreatedAt         time.Time      `db:"created_at"`
// 	UpdatedAt         sql.NullTime   `db:"updated_at"`
// }

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

	Sender    User `db:"sender"`
	Recipient User `db:"recipient"`
}
type TransactionDetails struct {
	ID              string       `db:"id"`
	ReferenceNumber string       `db:"reference_number"`
	Amount          float64      `db:"amount"`
	Status          string       `db:"status"`
	Description     string       `db:"description"`
	CreatedAt       sql.NullTime `db:"created_at"`

	// Sender details
	SenderID        string `db:"sender_id"`
	SenderFirstName string `db:"sender_first_name"`
	SenderLastName  string `db:"sender_last_name"`
	SenderWalletID  string `db:"sender_wallet_id"`
	SenderAccount   string `db:"sender_account_number"`

	// Recipient details
	RecipientID        string `db:"recipient_id"`
	RecipientFirstName string `db:"recipient_first_name"`
	RecipientLastName  string `db:"recipient_last_name"`
	RecipientWalletID  string `db:"recipient_wallet_id"`
	RecipientAccount   string `db:"recipient_account_number"`
}

const getTransactionBasicQuery = `SELECT 
			t.id, 
			t.reference_number, 
			t.status, 
			t.amount, 
			t.description, 
			t.created_at,

			-- Sender details
			su.id AS sender_id,
			su.first_name AS sender_first_name,
			su.last_name AS sender_last_name,
			s.id AS sender_wallet_id,
			s.account_number AS sender_account_number,

			-- Recipient details
			ru.id AS recipient_id,
			ru.first_name AS recipient_first_name,
			ru.last_name AS recipient_last_name,
			r.id AS recipient_wallet_id,
			r.account_number AS recipient_account_number

		FROM transactions t
		LEFT JOIN wallets s ON t.sender_wallet_id = s.id
		LEFT JOIN users su ON s.user_id = su.id  

		LEFT JOIN wallets r ON t.recipient_wallet_id = r.id
		LEFT JOIN users ru ON r.user_id = ru.id  

		
		`

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

func (db *DB) CreateTransaction(transaction *Transaction, tx *sql.Tx) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id string

	query := `
		INSERT INTO transactions (sender_wallet_id, recipient_wallet_id, amount, reference_number, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			transaction.SenderWalletID,
			transaction.RecipientWalletID,
			transaction.Amount,
			transaction.ReferenceNumber,
			transaction.Description,
		).Scan(&id)
		if err != nil {
			return "", err
		}
	} else {
		err := db.GetContext(ctx, &id, query,
			transaction.SenderWalletID,
			transaction.RecipientWalletID,
			transaction.Amount,
			transaction.ReferenceNumber,
			transaction.Description,
		)

		if err != nil {
			return "", err
		}
	}

	return id, nil
}

func (db *DB) UpdateTransactionStatus(transactionID string, status string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
        UPDATE transactions SET status=$1 WHERE id=$2`

	result, err := db.ExecContext(ctx, query, status, transactionID)
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

func (db *DB) GetTransaction(id string) (*TransactionDetails, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var transaction TransactionDetails

	query := getTransactionBasicQuery + `

		WHERE t.id = $1
	`

	err := db.GetContext(ctx, &transaction, query, id)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return &transaction, true, nil
}

type FilterTransactionsOptions struct {
	StartDate   *time.Time
	EndDate     *time.Time
	SearchQuery string
	Offset      int
	Limit       int
}

func (db *DB) GetTransactionsByWalletId(walletId string, option *FilterTransactionsOptions) ([]*TransactionDetails, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Base query to fetch transaction details
	query := getTransactionBasicQuery
	query += `WHERE (t.sender_wallet_id = $1 OR t.recipient_wallet_id = $2)`

	args := []interface{}{walletId, walletId}
	placeholderIdx := 3

	// Filter by start date if provided
	if option.StartDate != nil {
		query += " AND t.created_at >= $" + strconv.Itoa(placeholderIdx)
		args = append(args, option.StartDate)
		placeholderIdx++
	}

	// Filter by end date if provided
	if option.EndDate != nil {
		query += " AND t.created_at <= $" + strconv.Itoa(placeholderIdx)
		args = append(args, option.EndDate)
		placeholderIdx++
	}

	// Apply search filter across multiple fields
	if option.SearchQuery != "" {
		query += " AND ("

		// Search by reference number
		query += "t.reference_number ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		// Search by transaction status
		query += " OR t.status ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		// Search in sender details if sender_wallet_id is not the same as walletId
		query += " OR (t.sender_wallet_id != $1 AND (su.first_name ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		query += " OR su.last_name ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		query += " OR s.account_number ILIKE $" + strconv.Itoa(placeholderIdx) + "))"
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		// Search in recipient details if recipient_wallet_id is not the same as walletId
		query += " OR (t.recipient_wallet_id != $2 AND (ru.first_name ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		query += " OR ru.last_name ILIKE $" + strconv.Itoa(placeholderIdx)
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++

		query += " OR r.account_number ILIKE $" + strconv.Itoa(placeholderIdx) + ")))"
		args = append(args, "%"+option.SearchQuery+"%")
		placeholderIdx++
	}

	// Add sorting and limit for pagination
	query += " ORDER BY t.created_at DESC LIMIT $" + strconv.Itoa(placeholderIdx)
	args = append(args, option.Limit)
	placeholderIdx++

	// Add offset for pagination
	query += " OFFSET $" + strconv.Itoa(placeholderIdx)
	args = append(args, option.Offset)

	var transactions []*TransactionDetails
	err := db.SelectContext(ctx, &transactions, query, args...)
	if err != nil {
		return nil, false, err
	}

	if len(transactions) == 0 {
		return nil, false, nil
	}

	return transactions, true, nil
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
