package app

import (
	"net/http"

	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/middleware"
)

// This is where all our HTTP routes are defined
// Similar routes are grouped together and their respective constructors
// ...are called with the dependencies they need.
// Global level middlewares are used to wrap the routes.
// Route specific middlewares are used to wrap ONLY the routes that need them.
// It returns an http.Handler which can be used to start the server
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	middlewareRepo := middleware.New(app.errorHandler, app.Logger, app.DB, &app.Config)

	routeHandler := handler.NewRouteHandler(&handler.RouteHandler{
		DB:           app.DB,
		ErrHandler:   app.errorHandler,
		Config:       &app.Config,
		Mailer:       app.Mailer,
		Helper:       app.Helper,
		Kafka:        app.Kafka,
		FileUploader: app.FileUploader,
		Cache:        app.Cache,
	})

	// Health-check route
	mux.HandleFunc("GET /health", routeHandler.HandleHealthCheck)

	// Auth routes
	mux.HandleFunc("POST /auth/register", routeHandler.HandleAuthRegister)
	mux.HandleFunc("POST /auth/login", routeHandler.HandleAuthLogin)
	mux.HandleFunc("POST /auth/verify-account", routeHandler.HandleVerifyAccount)
	mux.HandleFunc("POST /auth/verify-account/resend", routeHandler.HandleResendVerificationOTP)
	mux.HandleFunc("POST /auth/forgot-password", routeHandler.HandleForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", routeHandler.HandleResetPassword)

	// Account routes
	mux.Handle("PATCH /account/pin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleSetAccountPin)))
	mux.Handle("GET /account/profile", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleUserProfile)))
	mux.Handle("PATCH /account/profile-picture", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleChangeProfilePicture)))
	mux.Handle("GET /account/next-of-kin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleGetNextOfKin)))
	mux.Handle("POST /account/next-of-kin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleAddNextOfKin)))

	// user KYC data  routes
	mux.Handle("POST /account/kyc/bvn", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleSaveUserBVN)))
	mux.Handle("POST /account/kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleSaveKYCData)))
	mux.Handle("GET /account/kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleGetAllUserKYCData)))

	// KYC routes
	mux.Handle("GET /kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleKYCs)))
	mux.Handle("GET /kyc/{id}", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleSingleYC)))

	// Wallet routes
	mux.Handle("GET /wallets", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleUserWallets)))
	mux.Handle("GET /wallets/{id}/details", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleWalletDetails)))
	mux.Handle("GET /wallets/{id}/balance", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleWalletBalance)))
	mux.Handle("GET /wallets/{id}/transactions", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleWalletTransactions)))

	// Transaction routes
	mux.Handle("POST /transactions/send-money", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleTransferMoney)))
	mux.Handle("GET /transactions/{id}", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(routeHandler.HandleTransactionDetails)))

	// utility routes
	mux.HandleFunc("POST /utility/upload-file", routeHandler.HandleUploadFile)

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
