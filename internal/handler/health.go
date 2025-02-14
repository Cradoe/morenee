package handler

import (
	"net/http"

	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/response"
)

type healthCheckHandler struct {
	err *errHandler.ErrorRepository
}

func NewHealthCheckHandler(err *errHandler.ErrorRepository) *healthCheckHandler {
	return &healthCheckHandler{
		err: err,
	}
}

// Provides a quick way to check if the service is up and running
func (app *healthCheckHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	message := "Up and grateful"

	err := response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		app.err.ServerError(w, r, err)
	}
}
