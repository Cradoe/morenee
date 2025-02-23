package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type KYCData struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	SubmissionData string    `db:"submission_data"`
	Verified       bool      `db:"verified"`
	CreatedAt      time.Time `db:"created_at"`
	RequirementID  string    `db:"kyc_requirement_id"`

	Requirement string `db:"requirement"`
}

type UserKycDataRepository interface {
	Insert(userID, submissionData, requirementID string) error
	GetAll(userID string) ([]KYCData, error)
	GetByRequirementId(userID, kycRequirementID string) (*KYCData, bool, error)
	UpgradeLevel(userID string) (bool, error)
}

type UserKycDataRepositoryImpl struct {
	db *sqlx.DB
}

func NewUserKycDataRepository(db *sqlx.DB) UserKycDataRepository {
	return &UserKycDataRepositoryImpl{db: db}
}

func (repo *UserKycDataRepositoryImpl) Insert(userID, submissionData, requirementID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		INSERT INTO user_kyc_data (user_id, submission_data, kyc_requirement_id)
		VALUES ($1, $2, $3)
	`

	_, err := repo.db.ExecContext(ctx, query, userID, submissionData, requirementID)
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserKycDataRepositoryImpl) GetAll(userID string) ([]KYCData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		SELECT 
			ukd.id, 
			ukd.user_id, 
			ukd.submission_data, 
			ukd.kyc_requirement_id,
			ukd.created_at, 
			ukd.verified, 
			kr.requirement
		FROM 
			user_kyc_data ukd
		LEFT JOIN 
			kyc_requirements kr 
		ON 
			ukd.kyc_requirement_id = kr.id
		WHERE 
			ukd.user_id = $1
	`

	rows, err := repo.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kycDataList []KYCData
	for rows.Next() {
		var kycData KYCData
		if err := rows.Scan(
			&kycData.ID,
			&kycData.UserID,
			&kycData.SubmissionData,
			&kycData.RequirementID,
			&kycData.CreatedAt,
			&kycData.Verified,
			&kycData.Requirement,
		); err != nil {
			return nil, err
		}
		kycDataList = append(kycDataList, kycData)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return kycDataList, nil
}

func (repo *UserKycDataRepositoryImpl) GetByRequirementId(userID, kycRequirementID string) (*KYCData, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		SELECT 
			id, 
			submission_data
		FROM 
			user_kyc_data
		WHERE 
			user_id = $1 AND kyc_requirement_id = $2
	`

	var kycData KYCData
	err := repo.db.GetContext(ctx, &kycData, query, userID, kycRequirementID)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return &kycData, true, nil
}

func (repo *UserKycDataRepositoryImpl) UpgradeLevel(userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Step 1: Get current KYC level of the user
	var currentLevelID int
	err := repo.db.QueryRowContext(ctx, "SELECT COALESCE(kyc_level_id, 0) FROM users WHERE id = $1", userID).Scan(&currentLevelID)
	if err != nil {
		return false, err
	}

	// Step 2: Check if all requirements are fulfilled for the current level
	query := `
		SELECT 
			klr.id
		FROM 
			kyc_requirements klr
		LEFT JOIN 
			user_kyc_data ukd 
		ON 
			klr.id = ukd.kyc_requirement_id 
			AND ukd.user_id = $1
		WHERE 
			klr.kyc_level_id = $2
			AND ukd.id IS NULL;
	`

	rows, err := repo.db.QueryContext(ctx, query, userID, currentLevelID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	// If there are unfulfilled requirements, return without upgrading
	if rows.Next() {
		return false, nil // User is not eligible for upgrade
	}

	// Step 3: If all requirements are met, upgrade to the next level
	upgradeQuery := `
		UPDATE 
			users 
		SET 
			kyc_level_id = (
				SELECT 
					id 
				FROM 
					kyc_levels 
				WHERE 
					id > $1 
				ORDER BY id ASC 
				LIMIT 1
			)
		WHERE 
			id = $2;
	`

	_, err = repo.db.ExecContext(ctx, upgradeQuery, currentLevelID, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}
