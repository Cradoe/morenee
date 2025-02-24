package models

import "time"

type KYCLevel struct {
	ID                  string                `db:"id"`
	LevelName           string                `db:"level_name"`
	DailyTransferLimit  float64               `db:"daily_transfer_limit"`
	WalletBalanceLimit  float64               `db:"wallet_balance_limit"`
	SingleTransferLimit float64               `db:"single_transfer_limit"`
	RequirementID       string                `db:"requirement_id"`
	Requirement         string                `db:"requirement"`
	Requirements        []KYCLevelRequirement `db:"requirements"`
}

type KYCLevelRequirement struct {
	ID          string `db:"id"`
	Requirement string `db:"requirement"`
}

type KYCData struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	SubmissionData string    `db:"submission_data"`
	Verified       bool      `db:"verified"`
	CreatedAt      time.Time `db:"created_at"`
	RequirementID  string    `db:"kyc_requirement_id"`

	Requirement string `db:"requirement"`
}
