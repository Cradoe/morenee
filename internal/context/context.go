package context

import (
	"context"
	"net/http"

	"github.com/cradoe/moremonee/internal/database"
)

type contextKey string

const (
	authenticatedUserContextKey = contextKey("authenticatedUser")
)

func ContextSetAuthenticatedUser(r *http.Request, user *database.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetAuthenticatedUser(r *http.Request) *database.User {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*database.User)
	if !ok {
		return nil
	}

	return user
}
