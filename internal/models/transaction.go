package models

import (
	"database/sql"
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
