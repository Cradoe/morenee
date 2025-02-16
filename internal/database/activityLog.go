// Logging is a critical part of the system
// Every action (synchronous or asynchronous) should be logged.
// This helps in audit and will also be used to trace activites.
// There's no such thing as too much log
// ...
// We used polymorphism to define entity and entity_id
// This allow our table to be used for different part of the application
// See https://..
package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ActivityLog struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	Entity      string    `db:"entity"`
	EntityId    string    `db:"entity_id"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// activity log entities
const (
	// ActivityLogTransactionEntity is used in actions that has to do with transactions and the transactions table
	ActivityLogTransactionEntity = "transaction"

	// ActivityLogWalletEntity is used in activites that has to do with wallets and the wallets table
	ActivityLogWalletEntity = "wallet"

	// ActivityLogUserEntity is used in activites that has to do with user account and the users table
	ActivityLogUserEntity = "user"
)

// possible descriptions
const (
	ActivityLogUserRegistrationDescription = "User registration"
	ActivityLogUserPinChangeDescription    = "User pin change"
	ActivityLogUserLoginDescription        = "User login"
	ActivityLogFailedLoginDescription      = "Failed login"

	ActivityLogTransactionInitiatedDescription   = "Transaction initiated"
	ActivityLogTransactionDebitDescription       = "Transaction debit"
	ActivityLogTransactionCreditDescription      = "Transaction credit"
	ActivityLogTransactionFailedDebitDescription = "Transaction debit failed"
	ActivityLogTransactionFailedDescription      = "Transaction failed"
	ActivityLogTransactionRevertedDescription    = "Transaction reverted"
	ActivityLogTransactionSuccessDescription     = "Transaction success"
)

func (db *DB) CreateActivityLog(log *ActivityLog) (*ActivityLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans ActivityLog

	query := `
		INSERT INTO account_logs (user_id, entity, entity_id, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := db.GetContext(ctx, &trans, query,
		log.UserID,
		log.Entity,
		log.EntityId,
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
func (db *DB) CountFailedLoginAttempts(user_id string) int {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var count int

	query := `SELECT count(user_id) FROM account_logs WHERE user_id = $1 AND entity = $2 AND description = $3`

	err := db.GetContext(ctx, &count, query, user_id, ActivityLogUserEntity, ActivityLogFailedLoginDescription)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}

	return count
}
