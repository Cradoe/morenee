package handler

import (
	dctx "context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/stream"
	"github.com/cradoe/morenee/internal/validator"
)

var (
	ErrInActiveRecipientAccount = errors.New("we can't process transfer into the recipient's account")
	ErrAttemptForSameAccount    = errors.New("your can't transfer to your own account")

	ErrInActiveSenderAccount       = errors.New("your account cannot process transaction at this time")
	ErrInsufficientBalance         = errors.New("insufficient balance")
	ErrDailyLimitExceeded          = errors.New("daily limit exceeded, upgrade your account")
	ErrSingleTransferLimitExceeded = errors.New("transfer limit exceeded, upgrade your account")
	ErrRecipientNotFound           = errors.New("recipient not found")
	ErrNoAccountPin                = errors.New("you need to set PIN for your account")
	ErrDuplicateTransfer           = errors.New("this appears to be a duplicate transaction")
	ErrInvalidPin                  = errors.New("invalid pin")
	ErrWalletNotFound              = errors.New("wallet not found")
)

const (
	transferDebitTopic = "transfer.debit"
)

type transactionHandler struct {
	db         *database.DB
	errHandler *errHandler.ErrorRepository
	kafka      *stream.KafkaStream
}

func NewTransactionHandler(db *database.DB, errHandler *errHandler.ErrorRepository, kafka *stream.KafkaStream) *transactionHandler {
	return &transactionHandler{
		db:         db,
		errHandler: errHandler,
		kafka:      kafka,
	}
}

type InitiatedTransfer struct {
	ID                string  `json:"id"`
	ReferenceNumber   string  `json:"reference_number"`
	SenderID          string  `json:"sender_id"`
	SenderWalletID    string  `json:"sender_wallet_id"`
	RecipientID       string  `json:"recipient_id"`
	RecipientWalletID string  `json:"recipient_wallet_id"`
	Status            string  `json:"status"`
	Amount            float64 `json:"amount"`
	CreatedAt         string  `json:"created_at"`
}

func (h *transactionHandler) HandleTransferMoney(w http.ResponseWriter, r *http.Request) {
	// To initiate a wallet to wallet transfer, we need to
	// Step 1: Verify account PIN
	// Step 2: Validate other input items and check for idempotency issue
	// Step 3: Account verifications, check activeness, daily limit, and co
	// Step 4. Perform quick lookups such as suspicious transfers, fraudulent activities, etc
	// Step 5: create a pending transaction and initialize a background worker to handle the rest

	type TransferFundsInput struct {
		AccountNumber   string              `json:"account_number"`
		Amount          float64             `json:"amount"`
		ReferenceNumber string              `json:"reference_number"`
		Description     string              `json:"description"`
		Pin             int                 `json:"pin"`
		Validator       validator.Validator `json:"-"`
	}

	var input TransferFundsInput

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.errHandler.BadRequest(w, r, err)
		return
	}

	// Step 1: Verify account PIN
	// This involves checking the user enters PIN,
	// has set PIN for their account, and
	// entered correct PIN

	input.Validator.Check(input.Pin > 0, "Pin is required")
	// we are intentionally returning early becauase we don't want to proceed futher if Pin is not given
	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	sender := context.ContextGetAuthenticatedUser(r)

	if !sender.Pin.Valid {
		// user has not set account pin
		input.Validator.AddError(ErrNoAccountPin.Error())
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}
	// check if pin is correct and return early if it's not
	if int(sender.Pin.Int32) != input.Pin {
		// user has not set account pin
		input.Validator.AddError(ErrInvalidPin.Error())
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	// Step 2: Validate other input items
	input.Validator.Check(input.Amount > 0, "Amount is required")

	input.Validator.Check(validator.NotBlank(input.ReferenceNumber), "Reference number is required")
	input.Validator.Check(validator.NotBlank(input.AccountNumber), "Recipient account number is required")

	if input.Validator.HasErrors() {
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	_, found, err := h.db.FindTransactionByReference(input.ReferenceNumber)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}
	// we intentionally don't want check this when we checked for the above
	// because we don't want the error message to be grouped together,
	// they have different nature.
	if found {
		input.Validator.AddError(ErrDuplicateTransfer.Error())
		h.errHandler.FailedValidation(w, r, input.Validator.Errors)
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
		recipientWallet, found, err := h.db.FindWalletByAccountNumber(input.AccountNumber)

		if !found {
			errCh <- fmt.Errorf("wallet_not_found")
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
		senderWallet, found, err := h.db.GetWalletDetails(sender.ID)
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
		if err.Error() == "wallet_not_found" {
			response.JSONErrorResponse(w, nil, ErrRecipientNotFound.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		h.errHandler.ServerError(w, r, err)
		return
	case recipientWallet = <-recipientCh:
	}

	select {
	case err := <-errCh:
		if err.Error() == "wallet_not_found" {
			response.JSONErrorResponse(w, nil, ErrWalletNotFound.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		h.errHandler.ServerError(w, r, err)
		return
	case senderWallet = <-senderCh:
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

	// check for single transfer limit
	if senderWallet.SingleTransferLimit < input.Amount {
		response.JSONErrorResponse(w, nil, ErrSingleTransferLimitExceeded.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Check for daily limit
	if exceeded, err := h.db.HasExceededDailyLimit(senderWallet.ID, input.Amount); err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	} else if exceeded {
		response.JSONErrorResponse(w, nil, ErrDailyLimitExceeded.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// Here, we can perform a quick lookups such as simple fraud alert, suspicious activities , etc
	// ...
	// skipping this because it involves machine learning
	// ...

	// Step 5: create a pending transaction and initialize a background worker to handle the rest
	log.Println("ddd", recipientWallet.ID)
	newTrans := &database.Transaction{
		SenderWalletID:    senderWallet.ID,
		RecipientWalletID: recipientWallet.ID,
		Amount:            input.Amount,
		ReferenceNumber:   input.ReferenceNumber,
		Description:       sql.NullString{String: input.Description, Valid: input.Description != ""},
	}
	transaction, err := h.db.CreateTransaction(newTrans, nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	message := "Transfer initiated successfully"

	transferRes := &InitiatedTransfer{
		ID:                transaction.ID,
		ReferenceNumber:   transaction.ReferenceNumber,
		SenderID:          sender.ID,
		RecipientID:       recipientWallet.UserID,
		SenderWalletID:    transaction.SenderWalletID,
		RecipientWalletID: transaction.RecipientWalletID,
		Status:            transaction.Status,
		Amount:            transaction.Amount,
		CreatedAt:         transaction.CreatedAt.Format(time.RFC3339),
	}

	jsonMessage, err := json.Marshal(transferRes)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	// Produce message so that the debit worker can debit the sender
	go h.kafka.ProduceMessage(transferDebitTopic, string(jsonMessage))

	go func() {
		_, err = h.db.CreateAccountLog(&database.AccountLog{
			UserID:      sender.ID,
			Entity:      database.AccountLogTransactionEntity,
			EntityId:    transaction.ID,
			Description: database.AccountLogTransactionInitiatedDescription,
		})

		if err != nil {
			log.Printf("Error logging transfer initiation action: %v", err)
		}
	}()

	err = response.JSONOkResponse(w, transferRes, message, nil)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}
}
