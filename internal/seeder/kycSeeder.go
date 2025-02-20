package seeders

import (
	"context"
	"database/sql"
	"log"
)

// seedKycData seeds KYC levels and their associated requirements
func (seeder *Seeder) seedKycData() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx, err := seeder.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	kycLevels := []struct {
		LevelName           string
		DailyTransferLimit  int
		WalletBalanceLimit  int
		SingleTransferLimit int
		Requirements        []string
	}{
		{
			LevelName:           "Tier 1",
			DailyTransferLimit:  50000,
			WalletBalanceLimit:  300000,
			SingleTransferLimit: 10000,
			Requirements:        []string{"Address", "BVN"},
		},
		{
			LevelName:           "Tier 2",
			DailyTransferLimit:  200000,
			WalletBalanceLimit:  500000,
			SingleTransferLimit: 100000,
			Requirements:        []string{"Government-issued ID"},
		},
		{
			LevelName:           "Tier 3",
			DailyTransferLimit:  5000000,
			WalletBalanceLimit:  1000000,
			SingleTransferLimit: 1000000,
			Requirements:        []string{"Proof of Address", "Occupation/Employer Information"},
		},
	}

	// Insert KYC levels and their requirements
	for _, level := range kycLevels {
		var kycLevelID string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO kyc_levels (level_name, daily_transfer_limit, wallet_balance_limit, single_transfer_limit) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (level_name) DO NOTHING
			RETURNING id;`,
			level.LevelName, level.DailyTransferLimit, level.WalletBalanceLimit, level.SingleTransferLimit,
		).Scan(&kycLevelID)

		// Check if the insert failed due to conflict (no ID returned)
		if err == sql.ErrNoRows {
			// Get the existing ID for the level to ensure requirements are still seeded
			err = tx.QueryRowContext(ctx, `SELECT id FROM kyc_levels WHERE level_name = $1`, level.LevelName).Scan(&kycLevelID)
		}

		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to insert or retrieve KYC level '%s': %v", level.LevelName, err)
		}

		// Insert the KYC requirements for the level
		for _, requirement := range level.Requirements {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO kyc_requirements (kyc_level_id, requirement) 
				VALUES ($1, $2) 
				ON CONFLICT DO NOTHING;`,
				kycLevelID, requirement,
			)
			if err != nil {
				tx.Rollback()
				log.Fatalf("Failed to insert KYC requirement '%s' for level '%s': %v", requirement, level.LevelName, err)
			}
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

}
