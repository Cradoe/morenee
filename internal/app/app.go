package app

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/env"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/file"
	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/smtp"
	"github.com/cradoe/morenee/internal/stream"
	"github.com/joho/godotenv"
)

// Essential services and resources are exposed to the application
// this makes it possible for methods to have access to these items and when they need them
type Application struct {
	Config       config.Config
	DB           *database.DB
	Logger       *slog.Logger
	Mailer       *smtp.Mailer
	WG           sync.WaitGroup
	errorHandler *errHandler.ErrorRepository
	helper       *helper.HelperRepository
	Kafka        *stream.KafkaStream
	FileUploader *file.FileUploader
}

func NewApplication(logger *slog.Logger) (*Application, error) {
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", "error", err)
	}

	var cfg config.Config

	// config values are loaded from the .env file
	// Default values are provided for these items and these should  strictly be values for development mode only
	// make sure no production-level value is exposed as default value here
	cfg.BaseURL = env.GetString("BASE_URL", "http://localhost:4444")
	cfg.HttpPort = env.GetInt("HTTP_PORT", 4444)

	cfg.Db.Dsn = env.GetString("DB_DSN", "user:pass@localhost:5432/db")
	cfg.Db.Automigrate = env.GetBool("DB_AUTOMIGRATE", true)

	cfg.Jwt.SecretKey = env.GetString("JWT_SECRET_KEY", "ajf5nx3qmp6zquevllxocxqvyz42ypuo")

	// server errors won't be sent via email if the NOTIFICATIONS_EMAIL wasn't set in the .env file
	cfg.Notifications.Email = env.GetString("NOTIFICATIONS_EMAIL", "")

	cfg.Smtp.Host = env.GetString("SMTP_HOST", "example.smtp.host")
	cfg.Smtp.Port = env.GetInt("SMTP_PORT", 25)
	cfg.Smtp.Username = env.GetString("SMTP_USERNAME", "example_username")
	cfg.Smtp.Password = env.GetString("SMTP_PASSWORD", "pa55word")
	cfg.Smtp.From = env.GetString("SMTP_FROM", "Example Name <no_reply@example.org>")

	cfg.KafkaServers = env.GetString("KAFKA_SERVERS", "localhost:9092")

	cfg.FileUploader.ApiKey = env.GetString("CLOUDINARY_API_KEY", "")
	cfg.FileUploader.CloudName = env.GetString("CLOUDINARY_CLOUD_NAME", "")
	cfg.FileUploader.ApiSecret = env.GetString("CLOUDINARY_API_SECRET", "")

	db, err := database.New(cfg.Db.Dsn, cfg.Db.Automigrate)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	mailer, err := smtp.NewMailer(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.From)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mailer: %w", err)
	}

	cfg.KafkaServers = env.GetString("KAFKA_SERVERS", "localhost:9092")

	helper := helper.New(&cfg.BaseURL)

	errorHandler := errHandler.New(cfg.Notifications.Email, mailer, logger, helper)

	kafkaStream := stream.New(cfg.KafkaServers)

	fileUploader := file.New(cfg.FileUploader.CloudName, cfg.FileUploader.ApiKey, cfg.FileUploader.ApiSecret)

	app := &Application{
		Config:       cfg,
		DB:           db,
		Logger:       logger,
		Mailer:       mailer,
		errorHandler: errorHandler,
		helper:       helper,
		Kafka:        kafkaStream,
		FileUploader: fileUploader,
	}

	return app, nil
}
