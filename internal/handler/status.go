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
	data := map[string]string{
		"Status": "OK",
	}

	err := response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.err.ServerError(w, r, err)
	}
}
