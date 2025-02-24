package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/cache"
	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/models"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/smtp"
	"github.com/cradoe/morenee/internal/validator"

	"github.com/cradoe/gopass"
	"github.com/pascaldekloe/jwt"
)

type AuthHandler struct {
	DB           *repository.DB
	UserRepo     repository.UserRepository
	ActivityRepo repository.ActivityRepository
	WalletRepo   repository.WalletRepository
	Config       *config.Config
	ErrHandler   *errHandler.ErrorHandler
	Mailer       smtp.MailerInterface
	Helper       *helper.Helper
	Cache        *cache.Cache
}

func NewAuthHandler(handler *AuthHandler) *AuthHandler {
	return &AuthHandler{
		DB:           handler.DB,
		UserRepo:     handler.UserRepo,
		ActivityRepo: handler.ActivityRepo,
		WalletRepo:   handler.WalletRepo,
		ErrHandler:   handler.ErrHandler,
		Config:       handler.Config,
		Mailer:       handler.Mailer,
		Helper:       handler.Helper,
		Cache:        handler.Cache,
	}
}

// New user registration typically involves:
// Input validations and checking that records has not already existed for the unique fields, such as enail
// We then start a database transaction to insert the user record and also create a wallet for the user
// Failed operatin at any point will make the function to rollback their actions
func (h *AuthHandler) HandleAuthRegister(w http.ResponseWriter, r *http.Request) {
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
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	// we need to validate the password to make sure it meets the minimum requirements
	// the Validate function returns a slice of errors if the password does not meet the requirements
	_, errs := gopass.Validate(input.Password)

	if errs != nil {
		// return any errors found before we check the other fields
		// It's important that users have a strong password
		h.ErrHandler.FailedValidation(w, r, errs)
		return
	}

	_, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
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
	found, err = h.UserRepo.CheckIfPhoneNumberExist(input.PhoneNumber)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	input.Validator.Check(!found, "Phone number has been registered")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	hashedPassword, err := gopass.Hash(input.Password)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// we are using transactions to make sure that if any of the operations fail
	// we can rollback the changes and return an error to the client
	// ...without having incomplete data in the operations
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	defer func() {
		// always make sure it rollback, if there is an error
		// ...and the transaction is not committed
		if err != nil {
			tx.Rollback()
		}
	}()

	createdUser := &models.User{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		PhoneNumber:    input.PhoneNumber,
		Gender:         input.Gender,
		HashedPassword: hashedPassword,
	}

	userID, err := h.UserRepo.Insert(createdUser, tx)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// send verification OTP
	h.Helper.BackgroundTask(r, func() error {
		createdUser.ID = userID
		localErr := h.generateAndSendVerificationOTP(createdUser)

		if localErr != nil {
			log.Printf("Error sending verification email: %v", localErr)
			return localErr
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
			UserID:      userID,
			Entity:      repository.ActivityLogUserEntity,
			EntityId:    userID,
			Description: UserActivityLogRegistrationDescription,
		})

		if localErr != nil {
			log.Printf("Error logging user registration action: %v", localErr)
			return localErr
		}

		return nil
	})

	message := "Account created successfully"

	err = response.JSONCreatedResponse(w, nil, message)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}

}

func (h *AuthHandler) generateAndSendVerificationOTP(user *models.User) error {

	otp, err := gopass.GenerateOTP(5)

	if err != nil {
		return err
	}

	// save the otp to cache
	cacheKey := "verify-account-otp:" + user.ID
	cacheExpiration := time.Hour

	err = h.Cache.Set(cacheKey, otp, cacheExpiration)
	if err != nil {
		return err
	}

	emailData := h.Helper.NewEmailData()
	emailData["Name"] = user.FirstName + " " + user.LastName
	emailData["OTP"] = otp
	emailData["OTPExpiration"] = cacheExpiration
	emailData["BankName"] = BankName

	err = h.Mailer.Send(user.Email, emailData, "verify-account.tmpl")
	if err != nil {
		log.Printf("Error verify account email: %v", err)
		return err
	}

	return nil
}

func (h *AuthHandler) HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	input.Validator.Check(validator.NotBlank(input.Password), "Password is required")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	user, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		message := "Incorrect email/password"
		response.JSONErrorResponse(w, nil, message, http.StatusUnauthorized, nil)
		return
	}

	// validate password is user is found

	passwordMatches, err := gopass.ComparePasswordAndHash(input.Password, user.HashedPassword)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !passwordMatches {

		h.Helper.BackgroundTask(r, func() error {
			_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
				UserID:      user.ID,
				Entity:      repository.ActivityLogUserEntity,
				EntityId:    user.ID,
				Description: UserActivityLogFailedLoginDescription,
			})

			if localErr != nil {
				log.Printf("Error logging failed login action: %v", localErr)
				return localErr
			}

			return nil
		})

		//  if password is not correct, log, that, and lock the account after 3 consecutive failed attempts
		count := h.ActivityRepo.CountConsecutiveFailedLoginAttempts(user.ID, UserActivityLogFailedLoginDescription)
		// check if we already have 2 failed login attempts before this one.
		if count >= 2 {
			h.Helper.BackgroundTask(r, func() error {
				localErr := h.UserRepo.Lock(user.ID)

				if localErr != nil {
					log.Printf("Error Locking account due to failed login action: %v", localErr)
					return localErr
				}

				return nil
			})

			h.Helper.BackgroundTask(r, func() error {
				_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
					UserID:      user.ID,
					Entity:      repository.ActivityLogUserEntity,
					EntityId:    user.ID,
					Description: UserActivityLogLockedAccountDescription,
				})

				if localErr != nil {
					log.Printf("Error logging failed login action: %v", localErr)
					return localErr
				}

				return nil
			})

			message := "Account has been locked. Please contact support"
			response.JSONErrorResponse(w, nil, message, http.StatusUnauthorized, nil)
			return
		}

		message := "Incorrect email/password"
		response.JSONErrorResponse(w, nil, message, http.StatusUnauthorized, nil)
		return
	}

	// check that account is active
	if user.Status == repository.UserAccountActivePending {
		message := "Account not yet verified"
		response.JSONErrorResponse(w, nil, message, http.StatusUnauthorized, nil)
		return
	}

	if user.Status == repository.UserAccountLockedStatus {
		message := "Account has been locked. Please contact support"
		response.JSONErrorResponse(w, nil, message, http.StatusUnauthorized, nil)
		return
	}

	h.Helper.BackgroundTask(r, func() error {

		_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
			UserID:      user.ID,
			Entity:      repository.ActivityLogUserEntity,
			EntityId:    user.ID,
			Description: UserActivityLogLoginDescription,
		})

		if localErr != nil {
			log.Printf("Error logging successful login action: %v", localErr)
			return localErr
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		emailData := h.Helper.NewEmailData()
		emailData["Name"] = user.FirstName + " " + user.LastName
		emailData["BankName"] = BankName

		localErr := h.Mailer.Send(user.Email, emailData, "login-alert.tmpl")
		if localErr != nil {
			log.Printf("Error sending login alert: %v", localErr)
			return localErr
		}

		return nil
	})

	var claims jwt.Claims
	claims.Subject = user.ID

	expiry := time.Now().Add(24 * time.Hour)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(expiry)

	claims.Issuer = h.Config.BaseURL
	claims.Audiences = []string{h.Config.BaseURL}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(h.Config.Jwt.SecretKey))

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	data := map[string]string{
		"auth_token":   string(jwtBytes),
		"token_expiry": expiry.Format(time.RFC3339),
	}
	message := "Login succesful"
	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}

}

func (h *AuthHandler) HandleVerifyAccount(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		OTP       string              `json:"otp"`
		Validator validator.Validator `json:"-"`
	}
	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	input.Validator.Check(validator.NotBlank(input.OTP), "OTP is required")

	user, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(found, "Email not recognized")
	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// check if account has already been verified
	if user.VerifiedAt.Valid {
		message := "Account already verified"
		response.JSONErrorResponse(w, nil, message, http.StatusBadRequest, nil)
		return
	}

	// get stored otp from cache
	cacheKey := "verify-account-otp:" + user.ID
	cacheExists, err := h.Cache.Exists(cacheKey)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !cacheExists {
		message := "Invalid/expired OTP"
		response.JSONErrorResponse(w, nil, message, http.StatusUnprocessableEntity, nil)
		return
	}

	storedOTP, err := h.Cache.Get(cacheKey)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	if storedOTP != input.OTP {
		message := "Invalid/expired OTP"
		response.JSONErrorResponse(w, nil, message, http.StatusUnprocessableEntity, nil)
		return
	}

	// we are using transactions to make sure that if any of the operations fail
	// we can rollback the changes and return an error to the client
	// ...without having incomplete data in the operations
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	defer func() {
		// always make sure it rollback, if there is an error
		// ...and the transaction is not committed
		if err != nil {
			tx.Rollback()
		}
	}()

	// update user account to verified
	err = h.UserRepo.Verify(user.ID, tx)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// generate a wallet for the created user
	walletHandler := NewWalletHandler(&WalletHandler{
		WalletRepo: h.WalletRepo,
	})
	wallet, err := walletHandler.generateWallet(user.ID, user.PhoneNumber, tx)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	h.Helper.BackgroundTask(r, func() error {
		emailData := h.Helper.NewEmailData()
		emailData["Name"] = user.FirstName + " " + user.LastName
		emailData["AccountNumber"] = wallet.AccountNumber
		emailData["BankName"] = BankName

		localErr := h.Mailer.Send(user.Email, emailData, "welcome-email.tmpl")
		if localErr != nil {
			log.Printf("Error send welcome email: %v", localErr)
			return localErr
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
			UserID:      user.ID,
			Entity:      repository.ActivityLogUserEntity,
			EntityId:    user.ID,
			Description: UserActivityLogAccountVerifiedDescription,
		})

		if localErr != nil {
			log.Printf("Error logging account verification action: %v", localErr)
			return localErr
		}

		return nil
	})

	message := "Account verified successfully"
	err = response.JSONOkResponse(w, nil, message, nil)
}

func (h *AuthHandler) HandleResendVerificationOTP(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Validator validator.Validator `json:"-"`
	}
	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	user, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(found, "Email not recognized")
	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// check if account has already been verified
	if user.VerifiedAt.Valid {
		message := "Account already verified"
		response.JSONErrorResponse(w, nil, message, http.StatusBadRequest, nil)
		return
	}

	err = h.generateAndSendVerificationOTP(user)

	if err != nil {
		log.Printf("Error sending verification email: %v", err)
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	message := "Verification OTP sent to your email"
	err = response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
}

func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Validator validator.Validator `json:"-"`
	}
	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	user, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(found, "Email not recognized")
	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	otp, err := gopass.GenerateOTP(5)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// save the otp to cache
	cacheKey := "forgot-password-otp:" + user.ID
	cacheExpiration := time.Second * 120
	err = h.Cache.Set(cacheKey, otp, cacheExpiration)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// send otp as email to user
	emailData := h.Helper.NewEmailData()
	emailData["Name"] = user.FirstName + " " + user.LastName
	emailData["OTP"] = otp
	emailData["OTPExpiration"] = cacheExpiration

	err = h.Mailer.Send(user.Email, emailData, "forgot-password.tmpl")
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	message := "OTP sent to your email"
	err = response.JSONOkResponse(w, nil, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

}

func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		OTP       string              `json:"otp"`
		Validator validator.Validator `json:"-"`
	}
	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.Email), "Email is required")
	input.Validator.Check(validator.IsEmail(input.Email), "Must be a valid email address")

	input.Validator.Check(validator.NotBlank(input.OTP), "OTP is required")

	// we need to validate the password to make sure it meets the minimum requirements
	// the Validate function returns a slice of errors if the password does not meet the requirements
	_, errs := gopass.Validate(input.Password)

	if errs != nil {
		// return any errors found before we check the other fields
		// It's important that users have a strong password
		h.ErrHandler.FailedValidation(w, r, errs)
		return
	}

	user, found, err := h.UserRepo.GetByEmail(input.Email)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	input.Validator.Check(found, "Email not recognized")
	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// get stored otp from cache
	cacheKey := "forgot-password-otp:" + user.ID

	cacheExists, err := h.Cache.Exists(cacheKey)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !cacheExists {
		message := "Invalid/expired OTP"
		response.JSONErrorResponse(w, nil, message, http.StatusUnprocessableEntity, nil)
		return
	}

	storedOTP, err := h.Cache.Get(cacheKey)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	if storedOTP != input.OTP {
		message := "Invalid/expired OTP"
		response.JSONErrorResponse(w, nil, message, http.StatusUnprocessableEntity, nil)
		return
	}

	hashedPassword, err := gopass.Hash(input.Password)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	err = h.UserRepo.UpdatePassword(user.ID, hashedPassword)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	h.Helper.BackgroundTask(r, func() error {
		emailData := h.Helper.NewEmailData()
		emailData["Name"] = user.FirstName + " " + user.LastName
		emailData["BankName"] = BankName

		localErr := h.Mailer.Send(user.Email, emailData, "password-reset.tmpl")
		if localErr != nil {
			log.Printf("Error sending password reset email: %v", localErr)
			return localErr
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		_, localErr := h.ActivityRepo.Insert(&models.ActivityLog{
			UserID:      user.ID,
			Entity:      repository.ActivityLogUserEntity,
			EntityId:    user.ID,
			Description: UserActivityLogPasswordResetDescription,
		})

		if localErr != nil {
			log.Printf("Error logging password reset action: %v", localErr)
			return localErr
		}

		return nil
	})

	message := "Password reset successfully"
	err = response.JSONOkResponse(w, nil, message, nil)

}
