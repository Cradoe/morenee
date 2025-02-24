package repository

import (
	"context"

	"github.com/cradoe/morenee/internal/models"
)

type KycRepository interface {
	GetAll() ([]models.KYCLevel, error)
	GetOne(id string) (*models.KYCLevel, bool, error)
}

type KycRepositoryImpl struct {
	db *DB
}

func NewKycRepository(db *DB) KycRepository {
	return &KycRepositoryImpl{db: db}
}

func (repo *KycRepositoryImpl) GetAll() ([]models.KYCLevel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		SELECT 
			kl.id, 
			kl.level_name, 
			kl.daily_transfer_limit, 
			kl.wallet_balance_limit, 
			kl.single_transfer_limit, 
			kr.id as requirement_id,
			kr.requirement
		FROM 
			kyc_levels kl
		LEFT JOIN 
			kyc_requirements kr 
		ON 
			kl.id = kr.kyc_level_id
		ORDER BY 
			kl.level_name;
	`

	rows, err := repo.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kycMap := make(map[string]*models.KYCLevel)

	for rows.Next() {
		var (
			levelID             string
			levelName           string
			dailyTransferLimit  float64
			walletBalanceLimit  float64
			singleTransferLimit float64
			requirementID       *string
			requirementValue    *string
		)

		if err := rows.Scan(
			&levelID,
			&levelName,
			&dailyTransferLimit,
			&walletBalanceLimit,
			&singleTransferLimit,
			&requirementID,
			&requirementValue,
		); err != nil {
			return nil, err
		}

		// Check if models.KYCLevel already exists in the map
		kyc, exists := kycMap[levelID]
		if !exists {
			// Create a new models.KYCLevel if it doesn't exist
			kyc = &models.KYCLevel{
				ID:                  levelID,
				LevelName:           levelName,
				DailyTransferLimit:  dailyTransferLimit,
				WalletBalanceLimit:  walletBalanceLimit,
				SingleTransferLimit: singleTransferLimit,
				Requirements:        []models.KYCLevelRequirement{},
			}
			kycMap[levelID] = kyc
		}

		// If a requirement is present, add it to the models.KYCLevel
		if requirementID != nil && requirementValue != nil {
			kyc.Requirements = append(kyc.Requirements, models.KYCLevelRequirement{
				ID:          *requirementID,
				Requirement: *requirementValue,
			})
		}
	}

	var kycLevels []models.KYCLevel
	for _, kyc := range kycMap {
		kycLevels = append(kycLevels, *kyc)
	}

	return kycLevels, nil
}
func (repo *KycRepositoryImpl) GetOne(id string) (*models.KYCLevel, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		SELECT 
			kl.id, 
			kl.level_name, 
			kl.daily_transfer_limit, 
			kl.wallet_balance_limit, 
			kl.single_transfer_limit, 
			kr.id as requirement_id,
			kr.requirement
		FROM 
			kyc_levels kl
		LEFT JOIN 
			kyc_requirements kr 
		ON 
			kl.id = kr.kyc_level_id
		WHERE 
			kl.id = $1;
	`

	rows, err := repo.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var kyc *models.KYCLevel
	kycRequirements := []models.KYCLevelRequirement{}

	for rows.Next() {
		var (
			requirementID    *string
			requirementValue *string
			tempKYC          models.KYCLevel
		)

		if err := rows.Scan(
			&tempKYC.ID,
			&tempKYC.LevelName,
			&tempKYC.DailyTransferLimit,
			&tempKYC.WalletBalanceLimit,
			&tempKYC.SingleTransferLimit,
			&requirementID,
			&requirementValue,
		); err != nil {
			return nil, false, err
		}

		if kyc == nil {
			kyc = &tempKYC
		}

		// Append requirement if it exists
		if requirementID != nil && requirementValue != nil {
			kycRequirements = append(kycRequirements, models.KYCLevelRequirement{
				ID:          *requirementID,
				Requirement: *requirementValue,
			})
		}
	}

	if kyc == nil {
		return nil, false, nil
	}

	kyc.Requirements = kycRequirements

	return kyc, true, nil
}
