package handler

import (
	"github.com/cradoe/morenee/internal/database"
)

// userHandler handles user-related requests
type userHandler struct {
	// server *Application
	db *database.DB
}

// NewUserHandler initializes a new UserHandler
func NewUserHandler(db *database.DB) *userHandler {
	return &userHandler{
		// server: server,
		db: db,
	}
}
