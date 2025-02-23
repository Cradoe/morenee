package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type NextOfKin struct {
	ID           string       `db:"id"`
	UserID       string       `db:"user_id"`
	Email        string       `db:"email"`
	FirstName    string       `db:"first_name"`
	LastName     string       `db:"last_name"`
	PhoneNumber  string       `db:"phone_number"`
	Relationship string       `db:"relationship"`
	Address      string       `db:"address"`
	CreatedAt    time.Time    `db:"created_at"`
	DeletedAt    sql.NullTime `db:"deleted_at"`
}

type NextOfKinRepository interface {
	Insert(nextOfKin *NextOfKin) (string, error)
	Update(id string, nextOfKin *NextOfKin) (bool, error)
	FindOneByUserID(userID string) (*NextOfKin, bool, error)
}

type NextOfKinRepositoryImpl struct {
	db *sqlx.DB
}

func NewNextOfKinRepository(db *sqlx.DB) NextOfKinRepository {
	return &NextOfKinRepositoryImpl{db: db}
}

func (repo *NextOfKinRepositoryImpl) Insert(nextOfKin *NextOfKin) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var id string

	query := `
		INSERT INTO  next_of_kins(user_id, first_name, last_name, phone_number, address, email, relationship )
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := repo.db.GetContext(ctx, &id, query,
		nextOfKin.UserID,
		nextOfKin.FirstName,
		nextOfKin.LastName,
		nextOfKin.PhoneNumber,
		nextOfKin.Address,
		nextOfKin.Email,
		nextOfKin.Relationship,
	)

	if err != nil {
		return "", err
	}

	return id, nil
}

func (repo *NextOfKinRepositoryImpl) Update(id string, nextOfKin *NextOfKin) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		UPDATE next_of_kins 
		SET first_name = $1, 
		    last_name = $2, 
		    phone_number = $3, 
		    address = $4, 
		    email = $5, 
		    relationship = $6
		WHERE id = $7
		`

	result, err := repo.db.ExecContext(ctx, query,
		nextOfKin.FirstName,
		nextOfKin.LastName,
		nextOfKin.PhoneNumber,
		nextOfKin.Address,
		nextOfKin.Email,
		nextOfKin.Relationship,
		id,
	)

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func (repo *NextOfKinRepositoryImpl) FindOneByUserID(userID string) (*NextOfKin, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var nextOfKin NextOfKin

	query := `SELECT id, first_name, last_name, email, address, phone_number, relationship, created_at 
	FROM next_of_kins WHERE user_id=$1 LIMIT 1`
	err := repo.db.GetContext(ctx, &nextOfKin, query, userID)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return &nextOfKin, true, nil
}
