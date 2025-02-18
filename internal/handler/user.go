package handler

import (
	"errors"
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

const (
	// UserActivityLogRegistrationDescription is used when a new user registers on the platform.
	UserActivityLogRegistrationDescription = "User registration"

	// UserActivityLogPinChangeDescription is used to log when a user changes their PIN for security purposes.
	UserActivityLogPinChangeDescription = "User pin change"

	// UserActivityLogLoginDescription is used when a user successfully logs into the platform.
	UserActivityLogLoginDescription = "User login"

	// UserActivityLogFailedLoginDescription is used when a login attempt fails, typically due to incorrect credentials.
	UserActivityLogFailedLoginDescription = "Failed login"

	// UserActivityLogLockedAccountDescription is used to log an activity where a user's account has been locked.
	// This log entry can be triggered due to multiple failed login attempts, security concerns, or manual actions by administrators.
	UserActivityLogLockedAccountDescription = "Locked account"
)

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
		_, err = h.db.CreateActivityLog(&database.ActivityLog{
			UserID:      user.ID,
			Entity:      database.ActivityLogUserEntity,
			EntityId:    user.ID,
			Description: UserActivityLogPinChangeDescription,
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
func (h *userHandler) HandleUserProfile(w http.ResponseWriter, r *http.Request) {

	user := context.ContextGetAuthenticatedUser((r))

	if user == nil {
		message := errors.New("unable to retrieve account details")
		h.errHandler.BadRequest(w, r, message)
		return
	}

	data := map[string]any{
		"id":           user.ID,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"email":        user.Email,
		"phone_number": user.PhoneNumber,
		"gender":       user.Gender,
		"created_at":   user.CreatedAt,
		"verified_at":  nil,
	}

	if user.VerifiedAt.Valid {
		data["verified_at"] = user.VerifiedAt.Time
	}

	message := "Profile fetched successfully"
	err := response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}
}
