// Logging is a critical part of the system
// Every action (synchronous or asynchronous) should be logged.
// This helps in audit and will also be used to trace activites.
// There's no such thing as too much log
// ...
// We used polymorphism to define type and type_id
// This allow our table to be used for different part of the application
// See https://..
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
	// AccountLogTypeTransaction is used in actions that has to do with transactions and the transactions table
	AccountLogTypeTransaction = "transaction"

	// AccountLogTypeWallet is used in activites that has to do with wallets and the wallets table
	AccountLogTypeWallet = "wallet"

	// AccountLogTypeWallet is used in activites that has to do with user account and the users table
	AccountLogTypeUser = "user"
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

// In order to prevent try-and-luck access into user's account
// ... we implement a feature to check for 3 consequtive failed login requests
// we can then temporarily lock the account for such occasion
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
