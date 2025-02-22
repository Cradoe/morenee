package handler

import (
	dctx "context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/validator"
)

var (
	ErrInActiveRecipientAccount    = errors.New("we can't process transfer into the recipient's account")
	ErrAttemptForSameAccount       = errors.New("your can't transfer to your own account")
	ErrTransactionDenied           = errors.New("transaction denied")
	ErrIncompatibleWalletCurrency  = errors.New("transaction can only happen between wallets of the same currency")
	ErrInActiveSenderAccount       = errors.New("your account cannot process transaction at this time")
	ErrInsufficientBalance         = errors.New("insufficient balance")
	ErrDailyLimitExceeded          = errors.New("daily limit exceeded, upgrade your account")
	ErrSingleTransferLimitExceeded = errors.New("transfer limit exceeded, upgrade your account")
	ErrCompleteProfileSetup        = errors.New("setup your bvn and address")
	ErrRecipientNotFound           = errors.New("recipient not found")
	ErrNoAccountPin                = errors.New("you need to set PIN for your account")
	ErrDuplicateTransfer           = errors.New("this appears to be a duplicate transaction")
	ErrInvalidPin                  = errors.New("invalid pin")
	ErrWalletNotFound              = errors.New("wallet not found")
	ErrInvalidStartDate            = errors.New("invalid start date format. Use YYYY-MM-DD")
	ErrInvalidEndDate              = errors.New("invalid end_date format. Use YYYY-MM-DD")
)

const (
	// TransactionActivityLogInitiatedDescription is used when a transaction is created and pending completion.
	TransactionActivityLogInitiatedDescription = "Transaction initiated"

	// TransactionActivityLogDebitDescription is used to log when a sender's wallet is debited successfully.
	TransactionActivityLogDebitDescription = "Transaction debit"

	// TransactionActivityLogCreditDescription is used to log when a recipient's wallet is credited successfully.
	TransactionActivityLogCreditDescription = "Transaction credit"

	// TransactionActivityLogFailedDebitDescription is used when a debit operation fails, potentially due to insufficient funds or errors.
	TransactionActivityLogFailedDebitDescription = "Transaction debit failed"

	// TransactionActivityLogFailedCreditDescription is used to log a failure to credit the recipient’s wallet or account.
	TransactionActivityLogFailedCreditDescription = "Transaction could not credit recipient"

	// TransactionActivityLogRevertedDescription is used when a previously failed transaction is reversed and the money is credited back to the sender.
	TransactionActivityLogRevertedDescription = "Transaction reverted"

	// TransactionActivityLogSuccessDescription is used to log the successful completion of a transaction.
	TransactionActivityLogSuccessDescription = "Transaction success"
)

const (
	transferDebitTopic = "transfer.debit"
)

type TransactionResponseData struct {
	ID              string             `json:"id"`
	ReferenceNumber string             `json:"reference_number"`
	Amount          float64            `json:"amount"`
	Description     string             `json:"description"`
	Status          string             `json:"status"`
	CreatedAt       time.Time          `json:"created_at"`
	Sender          MiniUserWithWallet `json:"sender"`
	Recipient       MiniUserWithWallet `json:"recipient"`
}

func (h *RouteHandler) HandleTransferMoney(w http.ResponseWriter, r *http.Request) {
	// To initiate a wallet to wallet transfer, we need to
	// Check idempotency key and return previous record if idempotency key is found in cache
	// Verify account PIN
	// Validate other input items and check for idempotency issue
	// Account verifications, check activeness, daily limit, and co
	// Perform quick lookups such as suspicious transfers, fraudulent activities, etc
	// reate a pending transaction and initialize a background worker to handle the rest

	type TransferFundsInput struct {
		SenderWalletID string              `json:"sender_wallet_id"`
		AccountNumber  string              `json:"account_number"`
		Amount         float64             `json:"amount"`
		Description    string              `json:"description"`
		Pin            int                 `json:"pin"`
		Validator      validator.Validator `json:"-"`
	}

	var input TransferFundsInput

	idempotencyKey := r.Header.Get("idempotency-key")
	if idempotencyKey == "" {
		message := "Invalid request"
		response.JSONErrorResponse(w, nil, message, http.StatusUnprocessableEntity, nil)
		return
	}

	// check if idempotencyKey exists in cache
	keyExists, err := h.Cache.Exists(idempotencyKey)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// if it does, let's get and return processed transaction details
	// this happens in cases like network retries
	if keyExists {
		previousResponse, err := h.Cache.Get(idempotencyKey)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
			return
		}

		var transferRes *TransactionResponseData

		// Convert string response from cache to bytes before unmarshaling
		err = json.Unmarshal([]byte(previousResponse), &transferRes)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
			return
		}

		// Since the transaction has already started processing,
		// we return details of the existing transaction
		message := "Transfer initiated successfully"
		err = response.JSONCreatedResponse(w, transferRes, message)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	err = request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	// Step 1: Verify account PIN
	// This involves checking the user enters PIN,
	// has set PIN for their account, and
	// entered correct PIN

	input.Validator.Check(input.Pin > 0, "Pin is required")
	// we are intentionally returning early becauase we don't want to proceed futher if Pin is not given
	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	sender := context.ContextGetAuthenticatedUser(r)

	if !sender.Pin.Valid {
		// user has not set account pin
		input.Validator.AddError(ErrNoAccountPin.Error())
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}
	// check if pin is correct and return early if it's not
	if int(sender.Pin.Int32) != input.Pin {
		// user has not set account pin
		input.Validator.AddError(ErrInvalidPin.Error())
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// Step 2: Validate other input items
	input.Validator.Check(input.Amount > 0, "Amount is required")

	input.Validator.Check(validator.NotBlank(input.SenderWalletID), "Sender wallet id is required")
	input.Validator.Check(validator.NotBlank(input.AccountNumber), "Recipient account number is required")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	ctx, cancel := dctx.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Step 3: Account verifications, this includes
	// activeness, daily limit, etc

	// we want to lookup sender's wallet and recipient's wallet in parallel
	// this will reduce waiting time for the client

	recipientCh := make(chan *database.Wallet, 1)
	senderCh := make(chan *database.Wallet, 1)
	errCh := make(chan error, 2)

	go func() {
		recipientWallet, found, err := h.DB.FindWalletByAccountNumber(input.AccountNumber)

		if !found {
			errCh <- fmt.Errorf("recipient_not_found")
			return
		}

		if err != nil {
			errCh <- err
			return
		}
		select {
		case recipientCh <- recipientWallet:
		case <-ctx.Done():
		}
	}()

	go func() {
		senderWallet, found, err := h.DB.GetWallet(input.SenderWalletID)
		if !found {
			errCh <- fmt.Errorf("wallet_not_found")
			return
		}

		if err != nil {
			errCh <- err
			return
		}
		select {
		case senderCh <- senderWallet:
		case <-ctx.Done():
		}
	}()

	var recipientWallet *database.Wallet
	var senderWallet *database.Wallet

	select {
	case err := <-errCh:
		if err.Error() == "recipient_not_found" {
			response.JSONErrorResponse(w, nil, ErrRecipientNotFound.Error(), http.StatusUnprocessableEntity, nil)
			return
		} else if err.Error() == "wallet_not_found" {
			response.JSONErrorResponse(w, nil, ErrWalletNotFound.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		h.ErrHandler.ServerError(w, r, err)
		return
	case recipientWallet = <-recipientCh:
	}

	select {
	case err := <-errCh:
		h.ErrHandler.ServerError(w, r, err)
		return
	case senderWallet = <-senderCh:
	}

	// check if logged in user is the owner of the wallet
	if sender.ID != senderWallet.UserID {
		response.JSONErrorResponse(w, nil, ErrTransactionDenied.Error(), http.StatusForbidden, nil)
		return
	}

	// check if it's an attempt to onself
	if recipientWallet.Currency != senderWallet.Currency {
		response.JSONErrorResponse(w, nil, ErrIncompatibleWalletCurrency.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// check if it's an attempt to onself
	if recipientWallet.AccountNumber == senderWallet.AccountNumber {
		response.JSONErrorResponse(w, nil, ErrAttemptForSameAccount.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Check sender wallet status
	if senderWallet.Status != database.WalletActiveStatus {
		response.JSONErrorResponse(w, nil, ErrInActiveSenderAccount.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Check recipient wallet status
	if recipientWallet.Status != database.WalletActiveStatus {
		response.JSONErrorResponse(w, nil, ErrInActiveRecipientAccount.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Check if sender has enough balance
	if senderWallet.Balance < input.Amount {
		response.JSONErrorResponse(w, nil, ErrInsufficientBalance.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// check sender kyc to be sure they are at least in kyc level 1
	var kycLevelIDStr string
	var senderKycLevel *database.KYCLevel
	if !sender.KYCLevelID.Valid {
		response.JSONErrorResponse(w, nil, ErrCompleteProfileSetup.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	kycLevelIDStr = fmt.Sprintf("%d", sender.KYCLevelID.Int16)

	level, kycLevelExists, err := h.DB.GetKYC(kycLevelIDStr)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
	if !kycLevelExists {
		response.JSONErrorResponse(w, nil, ErrCompleteProfileSetup.Error(), http.StatusUnprocessableEntity, nil)
		return
	}
	senderKycLevel = level

	// check sender kyc to be sure they can transfer this amount
	// check for single transfer limit
	if senderKycLevel.SingleTransferLimit < input.Amount {
		response.JSONErrorResponse(w, nil, ErrSingleTransferLimitExceeded.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Check for daily limit
	if exceeded, err := h.DB.HasExceededDailyLimit(senderWallet.ID, input.Amount, senderKycLevel.DailyTransferLimit); err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	} else if exceeded {
		response.JSONErrorResponse(w, nil, ErrDailyLimitExceeded.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Here, we can perform quick lookups such as simple fraud alert, suspicious activities , etc
	// ...
	// skipping this because it involves machine learning
	// ...

	// Step 5: create a pending transaction and initialize a background worker to handle the rest
	newTrans := &database.Transaction{
		SenderWalletID:    senderWallet.ID,
		RecipientWalletID: recipientWallet.ID,
		Amount:            input.Amount,
		ReferenceNumber:   generateTransactionRef(),
		Description:       sql.NullString{String: input.Description, Valid: input.Description != ""},
	}
	transactionId, err := h.DB.CreateTransaction(newTrans, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	transactionData, found, err := h.DB.GetTransaction(transactionId)
	if !found {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	transferRes := formTransactionResponseData(transactionData)

	jsonMessage, err := json.Marshal(&transferRes)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	// save idempotency key to cache for 10 minutes to prevent duplicate retries
	cacheKey := idempotencyKey
	cacheExpiration := time.Minute * 10

	err = h.Cache.Set(cacheKey, string(jsonMessage), cacheExpiration)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// Produce message so that the debit worker can debit the sender
	h.Helper.BackgroundTask(r, func() error {
		err := h.Kafka.ProduceMessage(transferDebitTopic, string(jsonMessage))
		if err != nil {
			log.Printf("Error producing message: %v", err)
			return err
		}

		return nil
	})

	h.Helper.BackgroundTask(r, func() error {
		_, err = h.DB.CreateActivityLog(&database.ActivityLog{
			UserID:      transferRes.Sender.ID,
			Entity:      database.ActivityLogTransactionEntity,
			EntityId:    transferRes.ID,
			Description: TransactionActivityLogInitiatedDescription,
		})

		if err != nil {
			log.Printf("Error logging transfer initiation action: %v", err)
			return err
		}

		return nil
	})

	message := "Transfer initiated successfully"
	err = response.JSONCreatedResponse(w, transferRes, message)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleWalletTransactions(w http.ResponseWriter, r *http.Request) {
	walletId := r.PathValue("id")

	var filterOptions = h.retrieveQueryValues(r)

	transactions, found, err := h.DB.GetTransactionsByWalletId(walletId, &database.FilterTransactionsOptions{
		StartDate:   filterOptions.StartDate,
		EndDate:     filterOptions.EndDate,
		SearchQuery: filterOptions.Search,
		Limit:       filterOptions.Limit,
		Offset:      filterOptions.Offset,
	})
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}
	if !found {
		message := "No transaction found"
		err = response.JSONOkResponse(w, []TransactionResponseData{}, message, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	message := "Transactions fetched successfully"

	data := make([]*TransactionResponseData, len(transactions))
	for i, t := range transactions {
		data[i] = formTransactionResponseData(t)
	}

	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleTransactionDetails(w http.ResponseWriter, r *http.Request) {
	transactionId := r.PathValue("id")

	transaction, found, err := h.DB.GetTransaction(transactionId)
	if !found {
		h.ErrHandler.NotFound(w, r)
		return
	}

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	result := formTransactionResponseData(transaction)

	message := "Details fetched successfully"

	err = response.JSONOkResponse(w, result, message, nil)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func formTransactionResponseData(transaction *database.TransactionDetails) *TransactionResponseData {
	return &TransactionResponseData{
		ID:              transaction.ID,
		ReferenceNumber: transaction.ReferenceNumber,
		Amount:          transaction.Amount,
		Status:          transaction.Status,
		Description:     transaction.Description,
		CreatedAt:       transaction.CreatedAt.Time,
		Sender: MiniUserWithWallet{
			ID:        transaction.SenderID,
			FirstName: transaction.SenderFirstName,
			LastName:  transaction.SenderLastName,
			Wallet: WalletMiniData{
				ID:            transaction.SenderWalletID,
				AccountNumber: transaction.SenderAccount,
				BankName:      BankName,
			},
		},
		Recipient: MiniUserWithWallet{
			ID:        transaction.RecipientID,
			FirstName: transaction.RecipientFirstName,
			LastName:  transaction.RecipientLastName,
			Wallet: WalletMiniData{
				ID:            transaction.RecipientWalletID,
				AccountNumber: transaction.RecipientAccount,
				BankName:      BankName,
			},
		},
	}
}

// GenerateTransactionRef generates a unique transaction reference
// Format: TX-{timestamp}-{randomHex}
func generateTransactionRef() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("TX-%d-%s", timestamp, randomHex)
}
