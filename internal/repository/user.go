package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/cradoe/morenee/internal/models"
)

type UserRepository interface {
	CheckIfPhoneNumberExist(phoneNumber string) (bool, error)
	Insert(user *models.User, tx *sql.Tx) (string, error)
	GetOne(id string) (*models.User, bool, error)
	GetByEmail(email string) (*models.User, bool, error)
	Verify(id string, tx *sql.Tx) error
	UpdatePassword(id, password string) error
	ChangePin(id string, pin string) error
	ChangeProfilePicture(id string, image string) error
	Lock(id string) error
}

const (
	// UserAccountActivePending indicates that the user's account has not been verified
	// This is the default status after registration
	UserAccountActivePending = "pending"

	// UserAccountActiveStatus indicates that the user's account is active and fully functional.
	// The user can log in, perform transactions, and access all account features.
	UserAccountActiveStatus = "active"

	// UserAccountLockedStatus indicates that the user's account has been locked.
	// This status may be used due to security reasons, such as multiple failed login attempts,
	// suspicious activity, or administrative action. A locked account cannot be accessed until unlocked.
	UserAccountLockedStatus = "locked"
)

type UserRepositoryImpl struct {
	db *DB
}

func NewUserRepository(db *DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

func (repo *UserRepositoryImpl) Insert(user *models.User, tx *sql.Tx) (string, error) {
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
		err := repo.db.GetContext(ctx, &id, query,
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

func (repo *UserRepositoryImpl) Verify(id string, tx *sql.Tx) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET status = $1, verified_at = $2 WHERE id = $3`

	if tx != nil {
		_, err := tx.ExecContext(ctx, query,
			UserAccountActiveStatus,
			time.Now(),
			id,
		)
		if err != nil {
			return err
		}
	} else {
		_, err := repo.db.ExecContext(ctx, query, UserAccountActiveStatus, time.Now(), id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (repo *UserRepositoryImpl) UpdatePassword(id, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET hashed_password = $1 WHERE id = $2`

	_, err := repo.db.ExecContext(ctx, query, password, id)
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserRepositoryImpl) GetOne(id string) (*models.User, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user models.User

	query := `SELECT * FROM users WHERE id = $1`

	err := repo.db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	return &user, true, err
}

func (repo *UserRepositoryImpl) GetByEmail(email string) (*models.User, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user models.User

	query := `SELECT * FROM users WHERE email = $1`

	err := repo.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	return &user, true, err
}
func (repo *UserRepositoryImpl) CheckIfPhoneNumberExist(phoneNumber string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)`

	err := repo.db.GetContext(ctx, &exists, query, phoneNumber)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (repo *UserRepositoryImpl) ChangePin(id string, pin string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET pin = $1 WHERE id = $2`

	_, err := repo.db.ExecContext(ctx, query, pin, id)
	return err
}

func (repo *UserRepositoryImpl) ChangeProfilePicture(id string, image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET image = $1 WHERE id = $2`

	_, err := repo.db.ExecContext(ctx, query, image, id)
	return err
}

func (repo *UserRepositoryImpl) Lock(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET status = $1 WHERE id = $2`

	_, err := repo.db.ExecContext(ctx, query, UserAccountLockedStatus, id)
	return err
}
