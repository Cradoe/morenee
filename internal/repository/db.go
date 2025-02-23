package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/cradoe/morenee/assets"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"
)

const defaultTimeout = 3 * time.Second

// Database interface defines available repositories
type Database interface {
	User() UserRepository
	Activity() ActivityRepository
	KYC() KycRepository
	UserKycData() UserKycDataRepository
	KycRequirement() KycRequirementRepository
	Transaction() TransactionRepository
	Wallet() WalletRepository
	NextOfKin() NextOfKinRepository

	Close() error
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// DatabaseImpl implements the Database interface
type DatabaseImpl struct {
	db                 *sqlx.DB
	userRepo           UserRepository
	activityRepo       ActivityRepository
	kycRepo            KycRepository
	kycDataRepo        UserKycDataRepository
	kycRequirementRepo KycRequirementRepository
	transactionRepo    TransactionRepository
	walletRepo         WalletRepository
	nextOfKinRepo      NextOfKinRepository

	mu sync.Mutex
}

// New initializes a database connection and runs migrations if enabled
func New(dsn string, automigrate bool) (Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "postgres", "postgres://"+dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(2 * time.Hour)

	// Run migrations if enabled
	if automigrate {
		iofsDriver, err := iofs.New(assets.EmbeddedFiles, "migrations")
		if err != nil {
			return nil, err
		}

		migrator, err := migrate.NewWithSourceInstance("iofs", iofsDriver, "postgres://"+dsn)
		if err != nil {
			return nil, err
		}

		if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return nil, err
		}
	}

	// Return DatabaseImpl instance without pre-initializing repositories
	return &DatabaseImpl{db: db}, nil
}

func (d *DatabaseImpl) Close() error {
	return d.db.Close()
}
func (d *DatabaseImpl) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	tx, err := d.db.BeginTxx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (d *DatabaseImpl) User() UserRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.userRepo == nil {
		d.userRepo = NewUserRepository(d.db)
	}
	return d.userRepo
}

func (d *DatabaseImpl) Activity() ActivityRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.activityRepo == nil {
		d.activityRepo = NewActivityRepository(d.db)
	}
	return d.activityRepo
}

func (d *DatabaseImpl) KYC() KycRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.kycRepo == nil {
		d.kycRepo = NewKycRepository(d.db)
	}
	return d.kycRepo
}

func (d *DatabaseImpl) UserKycData() UserKycDataRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.kycDataRepo == nil {
		d.kycDataRepo = NewUserKycDataRepository(d.db)
	}
	return d.kycDataRepo
}

func (d *DatabaseImpl) KycRequirement() KycRequirementRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.kycRequirementRepo == nil {
		d.kycRequirementRepo = NewKycRequirementRepository(d.db)
	}
	return d.kycRequirementRepo
}

func (d *DatabaseImpl) Transaction() TransactionRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.transactionRepo == nil {
		d.transactionRepo = NewTransactionRepository(d.db)
	}
	return d.transactionRepo
}

func (d *DatabaseImpl) Wallet() WalletRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.walletRepo == nil {
		d.walletRepo = NewWalletRepository(d.db)
	}
	return d.walletRepo
}

func (d *DatabaseImpl) NextOfKin() NextOfKinRepository {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.nextOfKinRepo == nil {
		d.nextOfKinRepo = NewNextOfKinRepository(d.db)
	}
	return d.nextOfKinRepo
}
