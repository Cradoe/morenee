package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cradoe/gopass"
	"github.com/cradoe/morenee/internal/context"
	database "github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/validator"
)

const (
	// UserActivityLogRegistrationDescription is used when a new user registers on the platform.
	UserActivityLogRegistrationDescription = "User registration"

	// UserActivityLogPinChangeDescription is used to log when a user changes their PIN for security purposes.
	UserActivityLogPinChangeDescription = "User pin change"

	// UserActivityLogAccountVerifiedDescription is used when user has verified their account
	UserActivityLogAccountVerifiedDescription = "User account verified"

	// UserActivityLogPasswordResetDescription is used when user reset their password
	UserActivityLogPasswordResetDescription = "User reset password"

	// UserActivityLogLoginDescription is used when a user successfully logs into the platform.
	UserActivityLogLoginDescription = "User login"

	// UserActivityLogFailedLoginDescription is used when a login attempt fails, typically due to incorrect credentials.
	UserActivityLogFailedLoginDescription = "Failed login"

	// UserActivityLogLockedAccountDescription is used to log an activity where a user's account has been locked.
	// This log entry can be triggered due to multiple failed login attempts, security concerns, or manual actions by administrators.
	UserActivityLogLockedAccountDescription = "Locked account"
)

type UserResponseData struct {
	ID          string           `json:"id"`
	FirstName   string           `json:"first_name"`
	LastName    string           `json:"last_name"`
	Email       string           `json:"email"`
	Image       string           `json:"image"`
	PhoneNumber string           `json:"phone_number"`
	Gender      string           `json:"gender"`
	CreatedAt   time.Time        `json:"created_at"`
	VerifiedAt  *time.Time       `json:"verified_at"`
	KYCLevel    *KYCResponseData `json:"kyc_level"`
}

type MiniUserWithWallet struct {
	ID        string         `json:"id"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Wallet    WalletMiniData `json:"wallet"`
}

func (h *RouteHandler) HandleSetAccountPin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Pin       string              `json:"pin"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	user := context.ContextGetAuthenticatedUser((r))

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Pin), "Pin is required")
	input.Validator.Check(validator.IsDigit(input.Pin), "Pin must be a 4 digit number")
	input.Validator.Check(len(input.Pin) == 4, "Pin must be a 4 digit number")

	input.Validator.Check(validator.NotBlank(input.Password), "Password is required")

	passwordMatches, err := gopass.ComparePasswordAndHash(input.Password, user.HashedPassword)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(passwordMatches, "Incorrect password")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	err = h.DB.User().ChangePin(user.ID, input.Pin)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	h.Helper.BackgroundTask(r, func() error {
		emailData := h.Helper.NewEmailData()
		emailData["Name"] = user.FirstName + " " + user.LastName
		emailData["BankName"] = BankName

		err = h.Mailer.Send(user.Email, emailData, "pin-changed.tmpl")
		if err != nil {
			log.Printf("sending pin changed action: %v", err)
			return err
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		_, err = h.DB.Activity().Insert(&database.ActivityLog{
			UserID:      user.ID,
			Entity:      database.ActivityLogUserEntity,
			EntityId:    user.ID,
			Description: UserActivityLogPinChangeDescription,
		})

		if err != nil {
			log.Printf("Error logging pin change action: %v", err)
			return err
		}

		return nil
	})

	message := "Pin set successfully"
	err = response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}

}
func (h *RouteHandler) HandleUserProfile(w http.ResponseWriter, r *http.Request) {

	user := context.ContextGetAuthenticatedUser((r))

	if user == nil {
		message := errors.New("unable to retrieve account details")
		h.ErrHandler.BadRequest(w, r, message)
		return
	}

	var verifiedAt *time.Time
	if user.VerifiedAt.Valid {
		verifiedAt = &user.VerifiedAt.Time
	}

	userResponse := UserResponseData{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Image:       user.Image.String,
		PhoneNumber: user.PhoneNumber,
		Gender:      user.Gender,
		CreatedAt:   user.CreatedAt,
		VerifiedAt:  verifiedAt,
	}

	var kycLevelIDStr string
	if user.KYCLevelID.Valid {
		kycLevelIDStr = fmt.Sprintf("%d", user.KYCLevelID.Int16)

		kycLevel, kycLevelExists, err := h.DB.KYC().GetOne(kycLevelIDStr)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}

		if kycLevelExists {
			userResponse.KYCLevel = &KYCResponseData{
				ID:        kycLevel.ID,
				LevelName: kycLevel.LevelName,
			}
		}
	}

	message := "Profile fetched successfully"
	err := response.JSONOkResponse(w, userResponse, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleChangeProfilePicture(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ImageUrl  string              `json:"image_url"`
		Validator validator.Validator `json:"-"`
	}

	user := context.ContextGetAuthenticatedUser((r))

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.ImageUrl), "File is required")
	input.Validator.Check(validator.IsURL(input.ImageUrl), "Image link must be a valid url")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	err = h.DB.User().ChangeProfilePicture(user.ID, input.ImageUrl)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	message := "Picture changed successfully"
	data := map[string]any{
		"image": input.ImageUrl,
	}
	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleGetNextOfKin(w http.ResponseWriter, r *http.Request) {

	user := context.ContextGetAuthenticatedUser((r))

	nextOfKin, found, err := h.DB.NextOfKin().FindOneByUserID(user.ID)

	if !found {
		h.ErrHandler.NotFound(w, r)
		return
	}

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	data := map[string]any{
		"id":           nextOfKin.ID,
		"first_name":   nextOfKin.FirstName,
		"last_name":    nextOfKin.LastName,
		"email":        nextOfKin.Email,
		"address":      nextOfKin.Address,
		"relationship": nextOfKin.Relationship,
		"phone_number": nextOfKin.PhoneNumber,
	}

	message := "Next of kin details fetched successfully."
	err = response.JSONOkResponse(w, data, message, nil)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleAddNextOfKin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email        string              `json:"email"`
		FirstName    string              `json:"first_name"`
		LastName     string              `json:"last_name"`
		PhoneNumber  string              `json:"phone_number"`
		Address      string              `json:"address"`
		Relationship string              `json:"relationship"`
		Validator    validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	input.Validator.Check(validator.NotBlank(input.FirstName), "First name is required")
	input.Validator.Check(validator.NotBlank(input.LastName), "Last name is required")
	input.Validator.Check(validator.NotBlank(input.PhoneNumber), "Phone numner is required")
	input.Validator.Check(validator.Matches(input.PhoneNumber, validator.RgxPhoneNumber), "Phone number must be in international format")
	input.Validator.Check(validator.NotBlank(input.Address), "Address is required")
	input.Validator.Check(validator.NotBlank(input.Relationship), "Relationship is required")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	user := context.ContextGetAuthenticatedUser((r))

	// check if user has previously added Next of kin
	existingRecord, found, _ := h.DB.NextOfKin().FindOneByUserID(user.ID)

	// if yes, then update the existing one
	if found {
		_, err = h.DB.NextOfKin().Update(existingRecord.ID, &database.NextOfKin{
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			Email:        input.Email,
			Address:      input.Address,
			PhoneNumber:  input.PhoneNumber,
			Relationship: input.Relationship,
		})

	} else {
		// create a new record
		_, err = h.DB.NextOfKin().Insert(&database.NextOfKin{
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			Email:        input.Email,
			Address:      input.Address,
			PhoneNumber:  input.PhoneNumber,
			Relationship: input.Relationship,
			UserID:       user.ID,
		})
	}

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	message := "Next of kin details saved successfully."
	err = response.JSONCreatedResponse(w, nil, message)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}
