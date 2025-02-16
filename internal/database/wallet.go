package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Wallet struct {
	ID                  string       `db:"id"`
	UserID              string       `db:"user_id"`
	Balance             float64      `db:"balance"`
	AccountNumber       string       `db:"account_number"`
	Currency            string       `db:"currency"`
	SingleTransferLimit float64      `db:"single_transfer_limit"`
	DailyTransferLimit  float64      `db:"daily_transfer_limit"`
	MaxBalance          float64      `db:"max_balance"`
	Status              string       `db:"status"`
	CreatedAt           time.Time    `db:"created_at"`
	DeletedAt           sql.NullTime `db:"deleted_at"`
	UpdatedAt           sql.NullTime `db:"updated_at"`
}

const (
	WalletActiveStatus = "active"
	WalletOnHoldStatus = "on-hold"
)

const (
	Level1SingleTransferLimit  float64 = 50_000
	Level1DailyTransferLimit   float64 = 200_000
	Level1WalletMaximumBalance float64 = 2_000_000
)

func (db *DB) CreateWallet(wallet *Wallet, tx *sql.Tx) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id string

	query := `
		INSERT INTO wallets (user_id, account_number, single_transfer_limit, daily_transfer_limit, max_balance)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			wallet.UserID,
			wallet.AccountNumber,
			Level1SingleTransferLimit,
			Level1DailyTransferLimit,
			Level1WalletMaximumBalance,
		).Scan(&id)
		if err != nil {
			return "", err
		}
	} else {
		err := db.GetContext(ctx, &id, query,
			wallet.UserID)

		if err != nil {
			return "", err
		}
	}

	return id, nil
}

func (db *DB) GetWalletBalance(id string) (*Wallet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT user_id, balance, currency FROM wallets WHERE id=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &wallet, nil
}

func (db *DB) GetWalletsByUserId(userID string) ([]Wallet, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallets []Wallet

	query := `
        SELECT id, balance, currency, account_number, status, single_transfer_limit, daily_transfer_limit, 
		max_balance, created_at FROM wallets WHERE user_id=$1 AND deleted_at IS NULL`

	err := db.SelectContext(ctx, &wallets, query, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return wallets, true, nil
}

func (db *DB) GetWallet(id string) (*Wallet, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT id, user_id, balance, currency, account_number, status, single_transfer_limit, daily_transfer_limit, 
		max_balance, created_at FROM wallets WHERE id=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &wallet, true, nil
}

func (db *DB) FindWalletByAccountNumber(account_number string) (*Wallet, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT id, user_id, balance, currency, account_number, status, created_at FROM wallets WHERE account_number=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, account_number)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &wallet, true, nil
}

func (db *DB) DebitWallet(walletID string, amount float64) (bool, error) {
	// we need to first check if the wallet has enough balance to process the transaction
	// if not, we return an error
	// if the wallet has enough balance, we proceed to debit the wallet
	// we'll use optimistic lock to lock the account for the duration of the operation

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}

	defer tx.Rollback()

	var wallet Wallet

	query := `
		SELECT balance FROM wallets WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`

	err = tx.GetContext(ctx, &wallet, query, walletID)

	if err != nil {
		return false, err
	}

	if wallet.Balance < amount {
		return false, nil
	}

	query = `
		UPDATE wallets SET balance=balance-$1 WHERE id=$2 AND deleted_at IS NULL`

	_, err = tx.ExecContext(ctx, query, amount, walletID)

	if err != nil {
		return false, err
	}

	err = tx.Commit()
	if err != nil {
		return false, err
	}

	return true, nil

}

func (db *DB) CreditWallet(walletID string, amount float64) (bool, error) {
	// we'll use optimistic lock to lock the account for the duration of the operation

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}

	defer tx.Rollback()

	var wallet Wallet

	query := `
		SELECT balance FROM wallets WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`

	err = tx.GetContext(ctx, &wallet, query, walletID)

	if err != nil {
		return false, err
	}

	query = `
		UPDATE wallets SET balance=balance+$1 WHERE id=$2 AND deleted_at IS NULL`

	_, err = tx.ExecContext(ctx, query, amount, walletID)

	if err != nil {
		return false, err
	}

	err = tx.Commit()
	if err != nil {
		return false, err
	}

	return true, nil

}

func (db *DB) LockWallet(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE wallets SET status = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, WalletOnHoldStatus, id)
	return err
}
