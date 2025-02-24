package repository

import (
	"context"
	"database/sql"
)

type KYCLevelRequirement struct {
	ID          string `db:"id"`
	Requirement string `db:"requirement"`
}

type KycRequirementRepository interface {
	FindByName(name string) (*KYCLevelRequirement, bool, error)
}

type KycRequirementRepositoryImpl struct {
	db *DB
}

func NewKycRequirementRepository(db *DB) KycRequirementRepository {
	return &KycRequirementRepositoryImpl{db: db}
}

func (repo *KycRequirementRepositoryImpl) FindByName(name string) (*KYCLevelRequirement, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var requirement KYCLevelRequirement
	query := `SELECT  id, requirement FROM kyc_requirements WHERE requirement = $1 LIMIT 1;`

	err := repo.db.GetContext(ctx, &requirement, query, name)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	return &requirement, true, nil
}
