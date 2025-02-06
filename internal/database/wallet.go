package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Wallet struct {
	ID                  int          `db:"id"`
	UserID              int          `db:"user_id"`
	Balance             float64      `db:"balance"`
	AccountNumber       string       `db:"account_number"`
	Currency            string       `db:"currency"`
	SingleTransferLimit float64      `db:"single_transfer_limit"`
	DailyTransferLimit  float64      `db:"daily_transfer_limit"`
	Status              string       `db:"status"`
	CreatedAt           time.Time    `db:"created_at"`
	DeletedAt           sql.NullTime `db:"deleted_at"`
	UpdatedAt           sql.NullTime `db:"updated_at"`
}

var (
	Level1SingleTransferLimit float64 = 50_000
	Level1DailyTransferLimit  float64 = 200_000
)

func (db *DB) CreateWallet(wallet *Wallet, tx *sql.Tx) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id int

	query := `
		INSERT INTO wallets (user_id, account_number, single_transfer_limit, daily_transfer_limit)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			wallet.UserID,
			wallet.AccountNumber,
			Level1SingleTransferLimit,
			Level1DailyTransferLimit,
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

func (db *DB) GetWalletBalance(userID int) (*Wallet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT balance, currency FROM wallets WHERE user_id=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &wallet, nil
}

func (db *DB) GetWalletDetails(userID int) (*Wallet, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT id, balance, currency, account_number, status, single_transfer_limit, daily_transfer_limit,
		 created_at FROM wallets WHERE user_id=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &wallet, true, nil
}

func (db *DB) GetWalletLimits(walletID int) (*Wallet, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var wallet Wallet

	query := `
        SELECT single_transfer_limit, daily_transfer_limit FROM wallets WHERE id=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, walletID)

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
        SELECT id, balance, currency, account_number, status, created_at FROM wallets WHERE account_number=$1 AND deleted_at IS NULL`

	err := db.GetContext(ctx, &wallet, query, account_number)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &wallet, true, nil
}
