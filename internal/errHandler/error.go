package errHandler

import (
	"fmt"
	"log/slog"
	"net/http"

	"runtime/debug"
	"strings"

	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/smtp"
)

type ErrorHandler struct {
	notificationEmail string
	logger            *slog.Logger
	mailer            *smtp.Mailer
}

func New(notificationEmail string, mailer *smtp.Mailer, logger *slog.Logger) *ErrorHandler {
	return &ErrorHandler{
		notificationEmail: notificationEmail,
		logger:            logger,
		mailer:            mailer,
	}
}

func (e *ErrorHandler) ReportServerError(r *http.Request, err error) {
	var (
		message = err.Error()
		method  string
		url     string
		trace   = string(debug.Stack())
	)

	// Check if r is nil before accessing its fields
	if r != nil {
		method = r.Method
		url = r.URL.String()
	} else {
		method = "Unknown"
		url = "Unknown"
	}

	// Log the error with request info
	requestAttrs := slog.Group("request", "method", method, "url", url)
	e.logger.Error(message, requestAttrs, "trace", trace)

	// Send notification email if required
	if e.notificationEmail != "" {
		data := map[string]any{
			"Message":       message,
			"RequestMethod": method,
			"RequestURL":    url,
			"Trace":         trace,
		}
		err := e.mailer.Send(e.notificationEmail, data, "error-notification.tmpl")
		if err != nil {
			trace = string(debug.Stack())
			e.logger.Error(err.Error(), requestAttrs, "trace", trace)
		}
	}
}

type Error struct {
	w       http.ResponseWriter
	r       *http.Request
	errors  any
	status  int
	message string
	headers http.Header
}

func (e *ErrorHandler) ErrorMessage(d *Error) {
	d.message = strings.ToUpper(d.message[:1]) + d.message[1:]

	err := response.JSONErrorResponse(d.w, d.errors, d.message, d.status, d.headers)
	if err != nil {
		e.ReportServerError(d.r, err)
		d.w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *ErrorHandler) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	e.ReportServerError(r, err)

	message := "The server encountered a problem and could not process your request"
	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusInternalServerError,
		message: message,
		headers: nil,
	})
}

func (e *ErrorHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"
	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusNotFound,
		message: message,
		headers: nil,
	})
}

func (e *ErrorHandler) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource", r.Method)
	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusMethodNotAllowed,
		message: message,
		headers: nil,
	})
}

func (e *ErrorHandler) BadRequest(w http.ResponseWriter, r *http.Request, err error) {

	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusBadRequest,
		message: err.Error(),
		headers: nil,
	})
}

func (e *ErrorHandler) FailedValidation(w http.ResponseWriter, r *http.Request, v any) {
	message := "Validation failed"

	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusUnprocessableEntity,
		message: message,
		headers: nil,
		errors:  v,
	})
}

func (e *ErrorHandler) InvalidAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	headers := make(http.Header)
	headers.Set("WWW-Authenticate", "Bearer")

	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusUnauthorized,
		message: "Invalid authentication token",
		headers: headers,
	})
}

func (e *ErrorHandler) AuthenticationRequired(w http.ResponseWriter, r *http.Request) {
	message := "You must be authenticated to access this resource"
	e.ErrorMessage(&Error{
		w:       w,
		r:       r,
		status:  http.StatusUnauthorized,
		message: message,
		headers: nil,
	})
}
