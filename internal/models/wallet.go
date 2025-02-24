package models

import (
	"database/sql"
	"time"
)

type Wallet struct {
	ID            string       `db:"id"`
	UserID        string       `db:"user_id"`
	Balance       float64      `db:"balance"`
	AccountNumber string       `db:"account_number"`
	Currency      string       `db:"currency"`
	Status        string       `db:"status"`
	CreatedAt     time.Time    `db:"created_at"`
	DeletedAt     sql.NullTime `db:"deleted_at"`
	UpdatedAt     sql.NullTime `db:"updated_at"`
}
