package database

import (
	"context"
	"database/sql"
	"time"
)

type Wallet struct {
	ID        int          `db:"id"`
	UserID    int          `db:"user_id"`
	Balance   float64      `db:"balance"`
	Currency  string       `db:"currency"`
	Status    string       `db:"status"`
	CreatedAt time.Time    `db:"created_at"`
	DeletedAt sql.NullTime `db:"deleted_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}

func (db *DB) CreateWallet(wallet *Wallet, tx *sql.Tx) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id int

	query := `
		INSERT INTO wallets (user_id)
		VALUES ($1)
		RETURNING id`
	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			wallet.UserID,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
	} else {
		err := db.GetContext(ctx, &id, query,
			wallet.UserID)

		if err != nil {
			return 0, err
		}
	}

	return id, nil
}
