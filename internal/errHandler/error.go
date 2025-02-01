package errHandler

import (
	"fmt"
	"log/slog"
	"net/http"

	"runtime/debug"
	"strings"

	"github.com/cradoe/gotemp/internal/helper"
	"github.com/cradoe/gotemp/internal/response"
	"github.com/cradoe/gotemp/internal/smtp"
	"github.com/cradoe/gotemp/internal/validator"
)

type ErrorRepository struct {
	notificationEmail string
	logger            *slog.Logger
	help              *helper.HelperRepository
	mailer            *smtp.Mailer
}

// NewErrorRepository initializes a new ErrorRepository
func NewErrorRepository(notificationEmail string, mailer *smtp.Mailer, logger *slog.Logger, help *helper.HelperRepository) *ErrorRepository {
	return &ErrorRepository{
		notificationEmail: notificationEmail,
		logger:            logger,
		help:              help,
		mailer:            mailer,
	}
}

func (e *ErrorRepository) ReportServerError(r *http.Request, err error) {
	var (
		message = err.Error()
		method  = r.Method
		url     = r.URL.String()
		trace   = string(debug.Stack())
	)

	requestAttrs := slog.Group("request", "method", method, "url", url)
	e.logger.Error(message, requestAttrs, "trace", trace)

	if e.notificationEmail != "" {
		data := e.help.NewEmailData()
		data["Message"] = message
		data["RequestMethod"] = method
		data["RequestURL"] = url
		data["Trace"] = trace

		err := e.mailer.Send(e.notificationEmail, data, "error-notification.tmpl")
		if err != nil {
			trace = string(debug.Stack())
			e.logger.Error(err.Error(), requestAttrs, "trace", trace)
		}
	}
}

func (e *ErrorRepository) ErrorMessage(w http.ResponseWriter, r *http.Request, status int, message string, headers http.Header) {
	message = strings.ToUpper(message[:1]) + message[1:]

	err := response.JSONWithHeaders(w, status, map[string]string{"Error": message}, headers)
	if err != nil {
		e.ReportServerError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *ErrorRepository) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	app.ReportServerError(r, err)

	message := "The server encountered a problem and could not process your request"
	app.ErrorMessage(w, r, http.StatusInternalServerError, message, nil)
}

func (e *ErrorRepository) NotFound(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"
	e.ErrorMessage(w, r, http.StatusNotFound, message, nil)
}

func (e *ErrorRepository) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource", r.Method)
	e.ErrorMessage(w, r, http.StatusMethodNotAllowed, message, nil)
}

func (e *ErrorRepository) BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	e.ErrorMessage(w, r, http.StatusBadRequest, err.Error(), nil)
}

func (e *ErrorRepository) FailedValidation(w http.ResponseWriter, r *http.Request, v validator.Validator) {
	err := response.JSON(w, http.StatusUnprocessableEntity, v)
	if err != nil {
		e.ServerError(w, r, err)
	}
}

func (e *ErrorRepository) InvalidAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	headers := make(http.Header)
	headers.Set("WWW-Authenticate", "Bearer")

	e.ErrorMessage(w, r, http.StatusUnauthorized, "Invalid authentication token", headers)
}

func (e *ErrorRepository) AuthenticationRequired(w http.ResponseWriter, r *http.Request) {
	e.ErrorMessage(w, r, http.StatusUnauthorized, "You must be authenticated to access this resource", nil)
}

func (e *ErrorRepository) BasicAuthenticationRequired(w http.ResponseWriter, r *http.Request) {
	headers := make(http.Header)
	headers.Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

	message := "You must be authenticated to access this resource"
	e.ErrorMessage(w, r, http.StatusUnauthorized, message, headers)
}
