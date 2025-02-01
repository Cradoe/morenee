package handler

import (
	"net/http"

	"github.com/cradoe/moremonee/internal/database"
	"github.com/cradoe/moremonee/internal/errHandler"
	"github.com/cradoe/moremonee/internal/password"
	"github.com/cradoe/moremonee/internal/request"
	"github.com/cradoe/moremonee/internal/validator"
)

type authHandler struct {
	db         *database.DB
	errHandler *errHandler.ErrorRepository
}

func NewAuthHandler(db *database.DB, errHandler *errHandler.ErrorRepository) *authHandler {
	return &authHandler{
		db:         db,
		errHandler: errHandler,
	}
}

func (h *authHandler) HandleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string              `json:"email"`
		Password    string              `json:"password"`
		FirstName   string              `json:"first_name"`
		LastName    string              `json:"last_name"`
		PhoneNumber string              `json:"phone_number"`
		Gender      string              `json:"gender"`
		Validator   validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.errHandler.BadRequest(w, r, err)
		return
	}

	_, found, err := h.db.GetUserByEmail(input.Email)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	input.Validator.CheckField(input.Email != "", "Email", "Email is required")
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "Email", "Must be a valid email address")
	input.Validator.CheckField(!found, "Email", "Email is already in use")

	input.Validator.CheckField(input.Password != "", "Password", "Password is required")
	input.Validator.CheckField(len(input.Password) >= 8, "Password", "Password is too short")
	input.Validator.CheckField(len(input.Password) <= 72, "Password", "Password is too long")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "Password", "Password is too common")

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator)
		return
	}

	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	newUser := &database.User{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		PhoneNumber:    input.PhoneNumber,
		Gender:         input.Gender,
		HashedPassword: hashedPassword,
	}

	_, err = h.db.InsertUser(newUser)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
