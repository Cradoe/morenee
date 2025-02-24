package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID             string         `db:"id"`
	KYCLevelID     sql.NullInt16  `db:"kyc_level_id"`
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
}
