package handler

import (
	"log"
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

	input.Validator.Check(input.Email != "", "Email is required")
	input.Validator.Check(validator.Matches(input.Email, validator.RgxEmail), "Must be a valid email address")
	input.Validator.Check(!found, "Email is already in use")

	ok, errs := gopass.Validate(input.Password)

	if !ok {
		h.errHandler.FailedValidation(w, r, errs)
		return
	}

	input.Validator.Check(input.FirstName != "", "First name is required")
	input.Validator.Check(len(input.FirstName) >= 3, "First name is too short")

	input.Validator.Check(input.LastName != "", "Last name is required")
	input.Validator.Check(len(input.LastName) >= 3, "Last name is too short")

	input.Validator.Check(input.Gender != "", "Gender is required")

	input.Validator.Check(input.PhoneNumber != "", "Phone number is required")
	input.Validator.Check(validator.Matches(input.PhoneNumber, validator.RgxPhoneNumber), "Phone number must be in international format")

	found, err = h.db.CheckIfPhoneNumberExist(input.PhoneNumber)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(!found, "Phone number has been registered")

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	hashedPassword, err := gopass.Hash(input.Password)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	newUser := &database.User{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		PhoneNumber:    input.PhoneNumber,
		Gender:         input.Gender,
		HashedPassword: hashedPassword,
	}

	userID, err := h.db.InsertUser(newUser, tx)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	// call the NewWalletHandler constructor and
	// then generate  a wallet for the created user
	walletHandler := NewWalletHandler(h.db, nil)
	_, err = walletHandler.generateWallet(userID, newUser.PhoneNumber, tx)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	// NB:: other operations that we could do include:
	// sending account activation email
	// but we are just going to skip that and focus on the core implementation of the system

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
	log.Println("foundfoundfound", found)
	input.Validator.Check(input.Email != "", "Email is required")
	input.Validator.Check(found, "Incorrect email/password")

	if found {
		passwordMatches, err := gopass.ComparePasswordAndHash(input.Password, user.HashedPassword)
		if err != nil {
			h.errHandler.ServerError(w, r, err)
			return
		}

		input.Validator.Check(input.Password != "", "Password is required")
		input.Validator.Check(passwordMatches, "Incorrect email/password")
	}

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
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
