package seeders

import (
	"time"

	"github.com/cradoe/morenee/internal/database"
)

const defaultTimeout = 5 * time.Second

type Seeder struct {
	DB *database.DB
}

func New(DB *database.DB) *Seeder {
	return &Seeder{
		DB: DB,
	}
}

func (seeder *Seeder) Run() {
	seeder.seedKycData()
}
