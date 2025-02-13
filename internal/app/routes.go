package app

import (
	"net/http"

	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/middleware"
)

func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	middlewareRepo := middleware.New(app.errorHandler, app.Logger, app.DB, &app.Config)
	healthHandler := handler.NewHealthCheckHandler(app.errorHandler)
	authHandler := handler.NewAuthHandler(app.DB, &app.Config, app.errorHandler)
	walletHandler := handler.NewWalletHandler(app.DB, app.errorHandler)
	transcHandler := handler.NewTransactionHandler(app.DB, app.errorHandler, app.Kafka)

	mux.HandleFunc("GET /health", healthHandler.HandleHealthCheck)

	mux.HandleFunc("POST /auth/register", authHandler.HandleAuthRegister)
	mux.HandleFunc("POST /auth/login", authHandler.HandleAuthLogin)

	mux.Handle("GET /wallet/balance", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletBalance)))
	mux.Handle("GET /wallet/details", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletDetails)))

	mux.Handle("POST /transactions/send-money", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(transcHandler.HandleTransferMoney)))

	return middlewareRepo.LogAccess(middlewareRepo.RecoverPanic(middlewareRepo.Authenticate(mux)))
}
