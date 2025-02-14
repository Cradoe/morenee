package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type AccountLog struct {
	ID          int       `db:"id"`
	UserID      int       `db:"user_id"`
	Type        string    `db:"type"`
	TypeId      int       `db:"type_id"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// account log types
const (
	AccountLogTypeTransaction = "transaction"
	AccountLogTypeWallet      = "wallet"
	AccountLogTypeUser        = "user"
)

// possible descriptions
const (
	AccountLogUserRegistrationDescription = "User registration"
	AccountLogUserPinChangeDescription    = "User pin change"
	AccountLogUserLoginDescription        = "User login"
	AccountLogFailedLoginDescription      = "Failed login"

	AccountLogTransactionInitiatedDescription = "Transaction initiated"
	AccountLogTransactionDebitDescription     = "Transaction debit"
	AccountLogTransactionCreditDescription    = "Transaction credit"
	AccountLogTransactionFailedDescription    = "Transaction failed"
	AccountLogTransactionRevertedDescription  = "Transaction reverted"
	AccountLogTransactionSuccessDescription   = "Transaction success"
)

func (db *DB) CreateAccountLog(log *AccountLog) (*AccountLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans AccountLog

	query := `
		INSERT INTO account_logs (user_id, type, type_id, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := db.GetContext(ctx, &trans, query,
		log.UserID,
		log.Type,
		log.TypeId,
		log.Description,
	)

	if err != nil {
		return nil, err
	}

	return &trans, nil
}

func (db *DB) CountFailedLoginAttempts(user_id int) int {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var count int

	query := `SELECT count(user_id) FROM account_logs WHERE user_id = $1 AND type = $2 AND description = $3`

	err := db.GetContext(ctx, &count, query, user_id, AccountLogTypeUser, AccountLogFailedLoginDescription)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}

	return count
}
