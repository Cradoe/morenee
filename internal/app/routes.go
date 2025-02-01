package app

import (
	"net/http"

	"github.com/cradoe/moremonee/internal/handler"
	"github.com/cradoe/moremonee/internal/middleware"
)

func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	middlewareRepo := middleware.NewMiddleware(app.errorHandler, app.Logger, app.DB, &app.Config)
	userHandler := handler.NewUserHandler(app.DB)
	healthHandler := handler.NewHealthCheckHandler(app.errorHandler)
	authHandler := handler.NewAuthHandler(app.DB, app.errorHandler)

	// mux.HandleFunc("GET /status", app.
	mux.HandleFunc("GET /health", healthHandler.HandleHealthCheck)
	mux.HandleFunc("POST /users", userHandler.HandleUsersCreate)

	mux.HandleFunc("POST /auth/register", authHandler.HandleAuthRegister)
	return middlewareRepo.LogAccess(middlewareRepo.RecoverPanic(middlewareRepo.Authenticate(mux)))
}
