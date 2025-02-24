package seeders

import (
	"time"

	"github.com/cradoe/morenee/internal/repository"
)

const defaultTimeout = 5 * time.Second

type Seeder struct {
	DB *repository.DB
}

func New(DB *repository.DB) *Seeder {
	return &Seeder{
		DB: DB,
	}
}

func (seeder *Seeder) Run() {
	seeder.seedKycData()
}
