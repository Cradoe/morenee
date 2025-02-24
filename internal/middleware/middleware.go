package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/response"

	"github.com/pascaldekloe/jwt"
	"github.com/tomasen/realip"
)

type Middleware struct {
	errHandler *errHandler.ErrorHandler
	logger     *slog.Logger
	UserRepo   repository.UserRepository
	config     *config.Config
}

func New(errHandler *errHandler.ErrorHandler, logger *slog.Logger, UserRepo repository.UserRepository, config *config.Config) *Middleware {
	return &Middleware{
		errHandler: errHandler,
		logger:     logger,
		UserRepo:   UserRepo,
		config:     config,
	}
}

func (mid *Middleware) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				mid.errHandler.ServerError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (mid *Middleware) LogAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := response.NewMetricsResponseWriter(w)
		next.ServeHTTP(mw, r)

		var (
			ip     = realip.FromRequest(r)
			method = r.Method
			url    = r.URL.String()
			proto  = r.Proto
		)

		userAttrs := slog.Group("user", "ip", ip)
		requestAttrs := slog.Group("request", "method", method, "url", url, "proto", proto)
		responseAttrs := slog.Group("repsonse", "status", mw.StatusCode, "size", mw.BytesCount)

		mid.logger.Info("access", userAttrs, requestAttrs, responseAttrs)
	})
}

func (mid *Middleware) Authenticate(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader != "" {
			headerParts := strings.Split(authorizationHeader, " ")

			if len(headerParts) == 2 && headerParts[0] == "Bearer" {
				token := headerParts[1]

				claims, err := jwt.HMACCheck([]byte(token), []byte(mid.config.Jwt.SecretKey))
				if err != nil {
					mid.errHandler.InvalidAuthenticationToken(w, r)
					return
				}

				if !claims.Valid(time.Now()) {
					mid.errHandler.InvalidAuthenticationToken(w, r)
					return
				}

				if claims.Issuer != mid.config.BaseURL {
					mid.errHandler.InvalidAuthenticationToken(w, r)
					return
				}

				if !claims.AcceptAudience(mid.config.BaseURL) {
					mid.errHandler.InvalidAuthenticationToken(w, r)
					return
				}

				userID := claims.Subject

				user, found, err := mid.UserRepo.GetOne(userID)
				if err != nil {
					mid.errHandler.ServerError(w, r, err)
					return
				}

				if found {
					r = context.ContextSetAuthenticatedUser(r, user)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (mid *Middleware) RequireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser := context.ContextGetAuthenticatedUser(r)

		if authenticatedUser == nil {
			mid.errHandler.AuthenticationRequired(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
