package handler

import (
	"net/http"
	"strconv"
	"time"

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

type queryStringValues struct {
	StartDate *time.Time
	EndDate   *time.Time
	Search    string
	Limit     int
	Offset    int
}

func (h *RouteHandler) retrieveQueryValues(r *http.Request) *queryStringValues {
	var queryValues = &queryStringValues{}

	// Parse start_date if provided
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr != "" {
		parsedStart, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			queryValues.StartDate = &parsedStart
		}
	}

	// Parse end_date if provided
	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr != "" {
		parsedEnd, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			queryValues.EndDate = &parsedEnd
		}
	}

	// Parse pagination params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("page")

	// Default pagination values
	offset := 0
	limit := 10

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	queryValues.Limit = limit

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 1 {
			offset = (parsedOffset - 1) * limit
		}
	}
	queryValues.Offset = offset

	// search params
	searchQuery := r.URL.Query().Get("search")
	queryValues.Search = searchQuery

	return queryValues
}
