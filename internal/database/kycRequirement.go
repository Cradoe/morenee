package database

import (
	"context"
	"database/sql"
)

type KYCLevelRequirement struct {
	ID          string `db:"id"`
	Requirement string `db:"requirement"`
}

func (db *DB) FindKYCRequirementByName(name string) (*KYCLevelRequirement, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var requirement KYCLevelRequirement
	query := `SELECT  id, requirement FROM kyc_requirements WHERE requirement = $1 LIMIT 1;`

	err := db.GetContext(ctx, &requirement, query, name)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	return &requirement, true, nil
}
