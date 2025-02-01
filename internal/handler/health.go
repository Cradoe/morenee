package handler

import (
	"net/http"

	"github.com/cradoe/gotemp/internal/errHandler"
	"github.com/cradoe/gotemp/internal/response"
)

type healthCheckHandler struct {
	err *errHandler.ErrorRepository
}

func NewHealthCheckHandler(err *errHandler.ErrorRepository) *healthCheckHandler {
	return &healthCheckHandler{
		err: err,
	}
}
func (app *healthCheckHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	message := "Up and grateful"

	err := response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		app.err.ServerError(w, r, err)
	}
}
