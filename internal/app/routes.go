package app

import (
	"net/http"

	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/middleware"
	"github.com/cradoe/morenee/internal/repository"
)

// This is where all our HTTP routes are defined
// Similar routes are grouped together and their respective constructors
// ...are called with the dependencies they need.
// Global level middlewares are used to wrap the routes.
// Route specific middlewares are used to wrap ONLY the routes that need them.
// It returns an http.Handler which can be used to start the server
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	// repositories
	userRepo := repository.NewUserRepository(app.DB)
	transactionRepo := repository.NewTransactionRepository(app.DB)
	activityRepo := repository.NewActivityRepository(app.DB)
	walletRepo := repository.NewWalletRepository(app.DB)
	kycRepo := repository.NewKycRepository(app.DB)
	nextOfKinRepo := repository.NewNextOfKinRepository(app.DB)
	kycRequirementRepo := repository.NewKycRequirementRepository(app.DB)
	userKycDataRepo := repository.NewUserKycDataRepository(app.DB)

	// middleware
	middlewareRepo := middleware.New(app.errorHandler, app.Logger, userRepo, &app.Config)

	// Health-check route
	routeHandler := handler.NewRouteHandler(&handler.RouteHandler{
		ErrHandler: app.errorHandler,
	})
	mux.HandleFunc("GET /health", routeHandler.HandleHealthCheck)

	// Auth routes
	authHandler := handler.NewAuthHandler(&handler.AuthHandler{
		DB:           app.DB,
		UserRepo:     userRepo,
		ActivityRepo: activityRepo,
		WalletRepo:   walletRepo,

		ErrHandler: app.errorHandler,
		Config:     &app.Config,
		Mailer:     app.Mailer,
		Helper:     app.Helper,
	})
	mux.HandleFunc("POST /auth/login", authHandler.HandleAuthLogin)
	mux.HandleFunc("POST /auth/register", authHandler.HandleAuthRegister)
	mux.HandleFunc("POST /auth/verify-account", authHandler.HandleVerifyAccount)
	mux.HandleFunc("POST /auth/verify-account/resend", authHandler.HandleResendVerificationOTP)
	mux.HandleFunc("POST /auth/forgot-password", authHandler.HandleForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", authHandler.HandleResetPassword)

	// Account routes
	userHandler := handler.NewUserHandler(&handler.UserHandler{
		UserRepo:      userRepo,
		ActivityRepo:  activityRepo,
		KycRepo:       kycRepo,
		NextOfKinRepo: nextOfKinRepo,

		ErrHandler: app.errorHandler,
		Mailer:     app.Mailer,
		Helper:     app.Helper,
	})
	mux.Handle("PATCH /account/pin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userHandler.HandleSetAccountPin)))
	mux.Handle("GET /account/profile", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userHandler.HandleUserProfile)))
	mux.Handle("PATCH /account/profile-picture", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userHandler.HandleChangeProfilePicture)))
	mux.Handle("GET /account/next-of-kin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userHandler.HandleGetNextOfKin)))
	mux.Handle("POST /account/next-of-kin", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userHandler.HandleAddNextOfKin)))

	// user KYC data  routes
	userKycDataHandler := handler.NewUserKycDataHandler(&handler.UserKycDataHandler{
		KycRequirementRepo: kycRequirementRepo,
		UserKycDataRepo:    userKycDataRepo,

		ErrHandler: app.errorHandler,
		Helper:     app.Helper,
	})
	mux.Handle("POST /account/kyc/bvn", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userKycDataHandler.HandleSaveUserBVN)))
	mux.Handle("POST /account/kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userKycDataHandler.HandleSaveKYCData)))
	mux.Handle("GET /account/kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(userKycDataHandler.HandleGetAllUserKYCData)))

	// KYC routes
	kycHandler := handler.NewKycHandler(&handler.KycHandler{
		KycRepo: kycRepo,

		ErrHandler: app.errorHandler,
	})
	mux.Handle("GET /kyc", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(kycHandler.HandleKYCs)))
	mux.Handle("GET /kyc/{id}", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(kycHandler.HandleSingleYC)))

	// Wallet routes
	walletHandler := handler.NewWalletHandler(&handler.WalletHandler{
		WalletRepo: walletRepo,

		ErrHandler: app.errorHandler,
	})
	mux.Handle("GET /wallets", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleUserWallets)))
	mux.Handle("GET /wallets/{id}/details", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletDetails)))
	mux.Handle("GET /wallets/{id}/balance", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(walletHandler.HandleWalletBalance)))

	// Transaction routes
	transactionHandler := handler.NewTransactionHandler(&handler.TransactionHandler{

		TransactionRepo: transactionRepo,
		WalletRepo:      walletRepo,
		ActivityRepo:    activityRepo,
		KycRepo:         kycRepo,

		ErrHandler: app.errorHandler,
		Helper:     app.Helper,
		Kafka:      app.Kafka,
	})
	mux.Handle("POST /transactions/send-money", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(transactionHandler.HandleTransferMoney)))
	mux.Handle("GET /transactions/{id}", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(transactionHandler.HandleTransactionDetails)))
	mux.Handle("GET /transactions/wallet/{id}/transactions", middlewareRepo.RequireAuthenticatedUser(http.HandlerFunc(transactionHandler.HandleWalletTransactions)))

	// utility routes
	utilityHandler := handler.NewUtilityHandler(&handler.UtilityHandler{
		FileUploader: app.FileUploader,
		ErrHandler:   app.errorHandler,
	})
	mux.HandleFunc("POST /utility/upload-file", utilityHandler.HandleUploadFile)

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
