// Logging is a critical part of the system
// Every action (synchronous or asynchronous) should be logged.
// This helps in audit and will also be used to trace activites.
// There's no such thing as too much log
// ...
// We used polymorphism to define entity and entity_id
// This allow our table to be used for different part of the application
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ActivityRepository interface {
	CountConsecutiveFailedLoginAttempts(userID, action_desc string) int
	Insert(log *ActivityLog) (*ActivityLog, error)
}

type ActivityLog struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	Entity      string    `db:"entity"`
	EntityId    string    `db:"entity_id"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

const (
	// ActivityLogTransactionEntity is used in actions that has to do with transactions and the transactions table
	ActivityLogTransactionEntity = "transaction"

	// ActivityLogWalletEntity is used in activites that has to do with wallets and the wallets table
	ActivityLogWalletEntity = "wallet"

	// ActivityLogUserEntity is used in activites that has to do with user account and the users table
	ActivityLogUserEntity = "user"
)

type ActivityRepositoryImpl struct {
	db *DB
}

func NewActivityRepository(db *DB) ActivityRepository {
	return &ActivityRepositoryImpl{db: db}
}

func (repo *ActivityRepositoryImpl) Insert(log *ActivityLog) (*ActivityLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var trans ActivityLog

	query := `
		INSERT INTO activity_logs (user_id, entity, entity_id, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := repo.db.GetContext(ctx, &trans, query,
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

// CountConsecutiveFailedLoginAttempts counts the number of consecutive failed login attempts for a user.
// This function is used to determine if a user’s account should be temporarily locked after 3 consecutive failures.
// It checks the most recent login attempts in descending order and counts failures until a successful login or the limit is reached.
func (repo *ActivityRepositoryImpl) CountConsecutiveFailedLoginAttempts(userID, action_desc string) int {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var descriptions []string

	// Query the most recent login attempts for the user, limiting to the last 3 entries
	query := `
		SELECT description 
		FROM activity_logs 
		WHERE user_id = $1 AND entity = $2 
		ORDER BY created_at DESC 
		LIMIT 3
	`
	err := repo.db.SelectContext(ctx, &descriptions, query, userID, ActivityLogUserEntity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		return 0
	}

	// Count consecutive failed logins
	count := 0
	for _, desc := range descriptions {
		if desc == action_desc {
			count++
		} else {
			break // Stop counting if we encounter a non-failed login
		}
	}

	return count
}
