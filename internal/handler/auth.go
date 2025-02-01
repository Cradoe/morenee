package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/validator"

	"github.com/cradoe/gopass"

	"github.com/pascaldekloe/jwt"
)

type authHandler struct {
	db         *database.DB
	config     *config.Config
	errHandler *errHandler.ErrorRepository
}

func NewAuthHandler(db *database.DB, config *config.Config, errHandler *errHandler.ErrorRepository) *authHandler {
	return &authHandler{
		db:         db,
		errHandler: errHandler,
		config:     config,
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
	input.Validator.CheckField(gopass.IsCommon(&input.Password), "Password", "Password is too common")

	input.Validator.CheckField(input.FirstName != "", "FirstName", "FirstName is required")
	input.Validator.CheckField(len(input.FirstName) >= 3, "FirstName", "FirstName is too short")

	input.Validator.CheckField(input.LastName != "", "LastName", "LastName is required")
	input.Validator.CheckField(len(input.LastName) >= 3, "LastName", "LastName is too short")

	input.Validator.CheckField(input.Gender != "", "Gender", "Gender is required")

	input.Validator.CheckField(input.PhoneNumber != "", "PhoneNumber", "PhoneNumber is required")
	input.Validator.CheckField(validator.Matches(input.PhoneNumber, validator.RgxPhoneNumber), "PhoneNumber", "Must be a valid phone number")

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator)
		return
	}

	hashedPassword, err := gopass.Hash(input.Password)
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

	message := "Account created successfully"
	err = response.JSONCreatedResponse(w, nil, message)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}

}

func (h *authHandler) HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.errHandler.BadRequest(w, r, err)
		return
	}

	user, found, err := h.db.GetUserByEmail(input.Email)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	input.Validator.CheckField(input.Email != "", "Email", "Email is required")
	input.Validator.CheckField(found, "Email", "Email address could not be found")

	if found {
		passwordMatches, err := gopass.Matches(input.Password, user.HashedPassword)
		if err != nil {
			h.errHandler.ServerError(w, r, err)
			return
		}

		input.Validator.CheckField(input.Password != "", "Password", "Password is required")
		input.Validator.CheckField(passwordMatches, "Password", "Password is incorrect")
	}

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator)
		return
	}

	var claims jwt.Claims
	claims.Subject = strconv.Itoa(user.ID)

	expiry := time.Now().Add(24 * time.Hour)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(expiry)

	claims.Issuer = h.config.BaseURL
	claims.Audiences = []string{h.config.BaseURL}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(h.config.Jwt.SecretKey))
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	data := map[string]string{
		"auth_token":   string(jwtBytes),
		"token_expiry": expiry.Format(time.RFC3339),
	}
	message := "Login succesful"
	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}

}
