package handler

import (
	"log"
	"net/http"

	"github.com/cradoe/gopass"
	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/validator"
)

type userHandler struct {
	db         *database.DB
	errHandler *errHandler.ErrorRepository
}

func NewUserHandler(db *database.DB, errHandler *errHandler.ErrorRepository) *userHandler {
	return &userHandler{
		db:         db,
		errHandler: errHandler,
	}
}

func (h *userHandler) HandleSetAccountPin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Pin       string              `json:"pin"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	user := context.ContextGetAuthenticatedUser((r))

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.errHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Pin), "Pin is required")
	input.Validator.Check(validator.IsDigit(input.Pin), "Pin must be a 4 digit number")
	input.Validator.Check(len(input.Pin) == 4, "Pin must be a 4 digit number")

	input.Validator.Check(validator.NotBlank(input.Password), "Password is required")

	passwordMatches, err := gopass.ComparePasswordAndHash(input.Password, user.HashedPassword)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(passwordMatches, "Incorrect password")

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	err = h.db.SetAccountPin(user.ID, input.Pin)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	go func() {
		_, err = h.db.CreateAccountLog(&database.AccountLog{
			UserID:      user.ID,
			Type:        database.AccountLogTypeUser,
			TypeId:      user.ID,
			Description: database.AccountLogUserPinChangeDescription,
		})

		if err != nil {
			log.Printf("Error logging pin change action: %v", err)
		}
	}()

	message := "Pin set successfully"
	err = response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}

}
