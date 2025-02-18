package handler

import (
	"net/http"

	"github.com/cradoe/morenee/internal/response"
)

// Provides a quick way to check if the service is up and running
func (app *RouteHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	message := "Up and grateful"

	err := response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		app.ErrHandler.ServerError(w, r, err)
	}
}
