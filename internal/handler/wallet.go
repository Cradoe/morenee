package handler

import (
	"database/sql"
	"net/http"

	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/response"
)

type walletHandler struct {
	db         *database.DB
	errHandler *errHandler.ErrorRepository
}

func NewWalletHandler(db *database.DB, errHandler *errHandler.ErrorRepository) *walletHandler {
	return &walletHandler{
		db:         db,
		errHandler: errHandler,
	}
}

func (h *walletHandler) generateWallet(user_id string, phone_number string, tx *sql.Tx) (bool, error) {

	// we don't have to manually check if account_number already exists because
	// we've established that phone_number is unique in users table.
	// However, if we, in the future, need to generate account number that's not user's phone number,
	// we'd have to validate non-existence.
	// We'll just keep it like this for now
	userWallet := &database.Wallet{
		UserID: user_id,
		AccountNumber: func() string {
			if len(phone_number) > 10 {
				return phone_number[len(phone_number)-10:]
			}
			return phone_number
		}(),
	}

	_, err := h.db.CreateWallet(userWallet, tx)
	if err != nil {
		return false, err
	}
	return true, nil

}

func (h *walletHandler) HandleWalletBalance(w http.ResponseWriter, r *http.Request) {
	user := context.ContextGetAuthenticatedUser((r))

	wallet, err := h.db.GetWalletBalance(user.ID)
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	message := "Balance fetched successfully"

	data := map[string]any{
		"balance":  wallet.Balance,
		"currency": wallet.Currency,
	}
	err = response.JSONOkResponse(w, data, message, nil)

	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}
}

func (h *walletHandler) HandleWalletDetails(w http.ResponseWriter, r *http.Request) {
	user := context.ContextGetAuthenticatedUser((r))

	wallet, found, err := h.db.GetWalletDetails(user.ID)
	if !found {
		response.JSONErrorResponse(w, nil, ErrWalletNotFound.Error(), http.StatusUnprocessableEntity, nil)
		return
	}
	if err != nil {
		h.errHandler.ServerError(w, r, err)
		return
	}

	message := "Wallet details successfully"

	data := map[string]any{
		"balance":        wallet.Balance,
		"currency":       wallet.Currency,
		"account_number": wallet.AccountNumber,
		"created_at":     wallet.CreatedAt,
	}
	err = response.JSONOkResponse(w, data, message, nil)

	if err != nil {
		h.errHandler.ServerError(w, r, err)
	}
}
