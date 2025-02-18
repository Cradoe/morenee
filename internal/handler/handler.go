package handler

import (
	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/file"
	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/smtp"
	"github.com/cradoe/morenee/internal/stream"
)

type RouteHandler struct {
	DB           *database.DB
	Config       *config.Config
	ErrHandler   *errHandler.ErrorRepository
	Mailer       *smtp.Mailer
	Helper       *helper.HelperRepository
	Kafka        *stream.KafkaStream
	FileUploader *file.FileUploader
}

func NewRouteHandler(handler *RouteHandler) *RouteHandler {
	return &RouteHandler{
		DB:           handler.DB,
		ErrHandler:   handler.ErrHandler,
		Config:       handler.Config,
		Mailer:       handler.Mailer,
		Helper:       handler.Helper,
		Kafka:        handler.Kafka,
		FileUploader: handler.FileUploader,
	}
}
