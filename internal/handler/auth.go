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

// New user registration typically involves:
// Input validations and checking that records has not already existed for the unique fields, such as enail
// We then start a database transaction to insert the user record and also create a wallet for the user
// Failed operatin at any point will make the function to rollback their actions
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

	// we need to validate the password to make sure it meets the minimum requirements
	// the Validate function returns a slice of errors if the password does not meet the requirements
	_, errs := gopass.Validate(input.Password)

	if errs != nil {
		// return any errors found before we check the other fields
		// It's important that users have a strong password
		h.errHandler.FailedValidation(w, r, errs)
		return
	}

	_, found, err := h.db.GetUserByEmail(input.Email)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	// we want to make sure no two users have the same email
	input.Validator.Check(!found, "Email is already in use")

	input.Validator.Check(validator.NotBlank(input.FirstName), "First name is required")
	input.Validator.Check(len(input.FirstName) >= 3, "First name is too short")

	input.Validator.Check(validator.NotBlank(input.LastName), "Last name is required")
	input.Validator.Check(len(input.LastName) >= 3, "Last name is too short")

	input.Validator.Check(validator.NotBlank(input.Gender), "Gender is required")

	input.Validator.Check(validator.NotBlank(input.PhoneNumber), "Phone number is required")
	input.Validator.Check(validator.Matches(input.PhoneNumber, validator.RgxPhoneNumber), "Phone number must be in international format")

	// we want to make sure no two users have the same phone number
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

	// we are using transactions to make sure that if any of the operations fail
	// we can rollback the changes and return an error to the client
	// ...without having incomplete data in the operations
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	defer func() {
		// always make sure it rollback, if there is an error
		// ...and the transaction is not committed
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

	go func() {
		_, err = h.db.CreateAccountLog(&database.AccountLog{
			UserID:      userID,
			Type:        database.AccountLogTypeUser,
			TypeId:      userID,
			Description: database.AccountLogUserRegistrationDescription,
		})

		if err != nil {
			log.Printf("Error logging user registration action: %v", err)
		}
	}()

	// NB:: other operations that we could do include:
	// sending account activation email
	// but we are just going to skip that and focus on the core implementation that
	// ... we are trying to achieve, which is to mock wallet-to-wallet transactions

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

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")
	input.Validator.Check(found, "Incorrect email/password")

	if found {
		passwordMatches, err := gopass.ComparePasswordAndHash(input.Password, user.HashedPassword)
		if err != nil {
			h.errHandler.ServerError(w, r, err)
			return
		}

		input.Validator.Check(validator.NotBlank(input.Password), "Password is required")
		input.Validator.Check(passwordMatches, "Incorrect email/password")

		if !passwordMatches {
			go func() {
				_, err = h.db.CreateAccountLog(&database.AccountLog{
					UserID:      user.ID,
					Type:        database.AccountLogTypeUser,
					TypeId:      user.ID,
					Description: database.AccountLogFailedLoginDescription,
				})

				if err != nil {
					log.Printf("Error logging failed login action: %v", err)
				}
			}()

			//  if password is not correct, log, that, and lock the account after 3 consecutive failed attempts
			count := h.db.CountFailedLoginAttempts(user.ID)
			// check if we already have 2 failed login attempts before this one.
			if count >= 2 {
				go h.db.UserLockAccount(user.ID)

				h.errHandler.FailedValidation(w, r, []string{"Account has been locked. Please contact support"})
				return
			}
		}

	}

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// check that account is active
	if user.Status != database.UserAccountActiveStatus {

		message := "Account has been locked. Please contact support"

		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)

		if err != nil {
			h.errHandler.ServerError(w, r, err)
		}
		return
	}

	go func() {
		_, err = h.db.CreateAccountLog(&database.AccountLog{
			UserID:      user.ID,
			Type:        database.AccountLogTypeUser,
			TypeId:      user.ID,
			Description: database.AccountLogUserLoginDescription,
		})

		if err != nil {
			log.Printf("Error logging successful login action: %v", err)
		}
	}()

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
