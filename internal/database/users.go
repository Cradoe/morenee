package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID             int          `db:"id"`
	FirstName      string       `db:"first_name"`
	LastName       string       `db:"last_name"`
	PhoneNumber    string       `db:"phone_number"`
	Gender         string       `db:"gender"`
	Email          string       `db:"email"`
	Status         string       `db:"status"`
	CreatedAt      time.Time    `db:"created_at"`
	DeletedAt      sql.NullTime `db:"deleted_at"`
	VerifiedAt     sql.NullTime `db:"verified_at"`
	HashedPassword string       `db:"hashed_password"`
}

func (db *DB) InsertUser(user *User, tx *sql.Tx) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id int
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
			return 0, err
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
			return 0, err
		}
	}

	return id, nil
}

func (db *DB) GetUser(id int) (*User, bool, error) {
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

func (db *DB) UpdateUserHashedPassword(id int, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET hashed_password = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, hashedPassword, id)
	return err
}
