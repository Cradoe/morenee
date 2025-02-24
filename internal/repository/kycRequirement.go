package repository

import (
	"context"
	"database/sql"

	"github.com/cradoe/morenee/internal/models"
)

type KycRequirementRepository interface {
	FindByName(name string) (*models.KYCLevelRequirement, bool, error)
}

type KycRequirementRepositoryImpl struct {
	db *DB
}

func NewKycRequirementRepository(db *DB) KycRequirementRepository {
	return &KycRequirementRepositoryImpl{db: db}
}

func (repo *KycRequirementRepositoryImpl) FindByName(name string) (*models.KYCLevelRequirement, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var requirement models.KYCLevelRequirement
	query := `SELECT  id, requirement FROM kyc_requirements WHERE requirement = $1 LIMIT 1;`

	err := repo.db.GetContext(ctx, &requirement, query, name)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	return &requirement, true, nil
}
