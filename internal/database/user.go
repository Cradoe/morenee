package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID             string         `db:"id"`
	FirstName      string         `db:"first_name"`
	LastName       string         `db:"last_name"`
	PhoneNumber    string         `db:"phone_number"`
	Image          sql.NullString `db:"image"`
	Gender         string         `db:"gender"`
	Email          string         `db:"email"`
	Status         string         `db:"status"`
	Pin            sql.NullInt32  `db:"pin"`
	CreatedAt      time.Time      `db:"created_at"`
	DeletedAt      sql.NullTime   `db:"deleted_at"`
	VerifiedAt     sql.NullTime   `db:"verified_at"`
	HashedPassword string         `db:"hashed_password"`

	Wallet Wallet `db:"wallet"`
}

const (
	// UserAccountActiveStatus indicates that the user's account is active and fully functional.
	// The user can log in, perform transactions, and access all account features.
	UserAccountActiveStatus = "active"

	// UserAccountLockedStatus indicates that the user's account has been locked.
	// This status may be used due to security reasons, such as multiple failed login attempts,
	// suspicious activity, or administrative action. A locked account cannot be accessed until unlocked.
	UserAccountLockedStatus = "locked"
)

func (db *DB) InsertUser(user *User, tx *sql.Tx) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id string
	query := `
		INSERT INTO users (first_name, last_name, phone_number, gender, email, hashed_password)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	if tx != nil {
		err := tx.QueryRowContext(ctx, query,
			user.FirstName,
			user.LastName,
			user.PhoneNumber,
			user.Gender,
			user.Email,
			user.HashedPassword,
		).Scan(&id)
		if err != nil {
			return "", err
		}
	} else {
		err := db.GetContext(ctx, &id, query,
			user.FirstName,
			user.LastName,
			user.PhoneNumber,
			user.Gender,
			user.Email,
			user.HashedPassword,
		)
		if err != nil {
			return "", err
		}
	}

	return id, nil
}

func (db *DB) GetUser(id string) (*User, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT * FROM users WHERE id = $1`

	err := db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	return &user, true, err
}

func (db *DB) GetUserByEmail(email string) (*User, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT * FROM users WHERE email = $1`

	err := db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	return &user, true, err
}
func (db *DB) CheckIfPhoneNumberExist(phone_number string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)`

	err := db.GetContext(ctx, &exists, query, phone_number)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (db *DB) UpdateUserHashedPassword(id string, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET hashed_password = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, hashedPassword, id)
	return err
}

func (db *DB) ChangeAccountPin(id string, pin string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET pin = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, pin, id)
	return err
}

func (db *DB) ChangeProfilePicture(id string, image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET image = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, image, id)
	return err
}

func (db *DB) UserLockAccount(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET status = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, UserAccountLockedStatus, id)
	return err
}
