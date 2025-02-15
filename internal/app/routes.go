package app

import (
	"net/http"

	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/middleware"
)

// This is where all our http routes are defined
// Similar routes are grouped together and their respective constructors
// ...are called to initiate them with whatever dependencies they need.
// Global level middlewares are used to wrap the routes.
// Route specific middlewares are used to wrap ONLY the routes that need them.
// It returns an http.Handler which can be used to start the server
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	middlewareRepo := middleware.New(app.errorHandler, app.Logger, app.DB, &app.Config)

	healthHandler := handler.NewHealthCheckHandler(app.errorHandler)
	mux.HandleFunc("GET /health", healthHandler.HandleHealthCheck)

	// Auth routes
	authHandler := handler.NewAuthHandler(app.DB, &app.Config, app.errorHandler)
	mux.HandleFunc("POST /auth/register", authHandler.HandleAuthRegister)
	mux.HandleFunc("POST /auth/set-pin", authHandler.HandleAuthRegister)
	mux.HandleFunc("POST /auth/login", authHandler.HandleAuthLogin)

	// Account routes
	accountHandler := handler.NewUserHandler(app.DB, app.errorHandler)
	mux.Handle("PATCH /account/pin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(accountHandler.HandleSetAccountPin)))

	// Wallet routes
	walletHandler := handler.NewWalletHandler(app.DB, app.errorHandler)
	mux.Handle("GET /wallet/balance", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletBalance)))
	mux.Handle("GET /wallet/details", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletDetails)))

	// Transaction routes
	transcHandler := handler.NewTransactionHandler(app.DB, app.errorHandler, app.Kafka)
	mux.Handle("POST /transactions/send-money", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(transcHandler.HandleTransferMoney)))

	// we need to handle all other routes that are not defined in the mux.
	// This is when user tries to access a route that does not exist
	//  We define a catch-all route
	// ...which returns a 404 not found error
	// ...to the client
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.errorHandler.NotFound(w, r)
	})

	return middlewareRepo.LogAccess(middlewareRepo.RecoverPanic(middlewareRepo.Authenticate(mux)))
}
