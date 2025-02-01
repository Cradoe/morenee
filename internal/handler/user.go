package handler

import (
	"net/http"

	"github.com/cradoe/gotemp/internal/database"
	"github.com/cradoe/gotemp/internal/request"
	"github.com/cradoe/gotemp/internal/validator"
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

func (h *userHandler) HandleUsersCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"Email"`
		Password  string              `json:"Password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		// h.server.badRequest(w, r, err)
		return
	}

	_, found, err := h.db.GetUserByEmail(input.Email)
	if err != nil {
		// h.server.serverError(w, r, err)
		return
	}

	input.Validator.CheckField(input.Email != "", "Email", "Email is required")
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "Email", "Must be a valid email address")
	input.Validator.CheckField(!found, "Email", "Email is already in use")

	input.Validator.CheckField(input.Password != "", "Password", "Password is required")
	input.Validator.CheckField(len(input.Password) >= 8, "Password", "Password is too short")
	input.Validator.CheckField(len(input.Password) <= 72, "Password", "Password is too long")

	if input.Validator.HasErrors() {
		// h.server.failedValidation(w, r, input.Validator)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
