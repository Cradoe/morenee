// context are used to save values in session for the lifecyle of the Request
// we set value at the top-layer, which can then be retrieved
// later in the application
// Typically used in middlewares and handlers
package context

import (
	"context"
	"net/http"

	database "github.com/cradoe/morenee/internal/repository"
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
