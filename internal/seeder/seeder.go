package seeders

import (
	"time"

	"github.com/cradoe/morenee/internal/repository"
	database "github.com/cradoe/morenee/internal/repository"
)

const defaultTimeout = 5 * time.Second

type Seeder struct {
	DB repository.Database
}

func New(DB database.Database) *Seeder {
	return &Seeder{
		DB: DB,
	}
}

func (seeder *Seeder) Run() {
	seeder.seedKycData()
}
