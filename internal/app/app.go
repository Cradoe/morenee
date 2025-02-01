package app

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/cradoe/moremonee/internal/config"
	"github.com/cradoe/moremonee/internal/database"
	"github.com/cradoe/moremonee/internal/env"
	"github.com/cradoe/moremonee/internal/errHandler"
	"github.com/cradoe/moremonee/internal/helper"
	"github.com/cradoe/moremonee/internal/smtp"
	"github.com/joho/godotenv"
)

type Application struct {
	Config       config.Config
	DB           *database.DB
	Logger       *slog.Logger
	Mailer       *smtp.Mailer
	WG           sync.WaitGroup
	errorHandler *errHandler.ErrorRepository
	helper       *helper.HelperRepository
}

// NewApplication initializes a new Application instance
func NewApplication(logger *slog.Logger) (*Application, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", "error", err)
	}

	var cfg config.Config

	cfg.BaseURL = env.GetString("BASE_URL", "http://localhost:4444")
	cfg.HttpPort = env.GetInt("HTTP_PORT", 4444)
	cfg.Db.Dsn = env.GetString("DB_DSN", "user:pass@localhost:5432/db")
	cfg.Db.Automigrate = env.GetBool("DB_AUTOMIGRATE", true)
	cfg.Jwt.SecretKey = env.GetString("JWT_SECRET_KEY", "ajf5nx3qmp6zquevllxocxqvyz42ypuo")
	cfg.Notifications.Email = env.GetString("NOTIFICATIONS_EMAIL", "")
	cfg.Smtp.Host = env.GetString("SMTP_HOST", "example.smtp.host")
	cfg.Smtp.Port = env.GetInt("SMTP_PORT", 25)
	cfg.Smtp.Username = env.GetString("SMTP_USERNAME", "example_username")
	cfg.Smtp.Password = env.GetString("SMTP_PASSWORD", "pa55word")
	cfg.Smtp.From = env.GetString("SMTP_FROM", "Example Name <no_reply@example.org>")

	// Initialize Database
	db, err := database.New(cfg.Db.Dsn, cfg.Db.Automigrate)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Mailer
	mailer, err := smtp.NewMailer(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.From)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mailer: %w", err)
	}

	helperRepository := helper.NewHelperRepository(&cfg.BaseURL)
	errorRepository := errHandler.NewErrorRepository(cfg.Notifications.Email, mailer, logger, helperRepository)
	app := &Application{
		Config:       cfg,
		DB:           db,
		Logger:       logger,
		Mailer:       mailer,
		errorHandler: errorRepository,
		helper:       helperRepository,
	}

	return app, nil
}
