// context are used to save values in session for the lifecyle of the Request
// we set value at the top-layer, which can then be retrieved
// later in the application
// Typically used in middlewares and handlers
package context

import (
	"context"
	"net/http"

	"github.com/cradoe/morenee/internal/models"
)

type contextKey string

const (
	authenticatedUserContextKey = contextKey("authenticatedUser")
)

func ContextSetAuthenticatedUser(r *http.Request, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetAuthenticatedUser(r *http.Request) *models.User {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*models.User)
	if !ok {
		return nil
	}

	return user
}
