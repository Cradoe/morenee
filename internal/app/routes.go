package app

import (
	"net/http"

	"github.com/cradoe/gotemp/internal/handler"
	"github.com/cradoe/gotemp/internal/middleware"
)

func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	middlewareRepo := middleware.NewMiddleware(app.errorHandler, app.Logger, app.DB, &app.Config)
	userHandler := handler.NewUserHandler(app.DB)
	healthHandler := handler.NewHealthCheckHandler(app.errorHandler)

	// mux.HandleFunc("GET /status", app.
	mux.HandleFunc("GET /status", healthHandler.HandleHealthCheck)
	mux.HandleFunc("POST /users", userHandler.HandleUsersCreate)

	return middlewareRepo.LogAccess(middlewareRepo.RecoverPanic(middlewareRepo.Authenticate(mux)))
}
