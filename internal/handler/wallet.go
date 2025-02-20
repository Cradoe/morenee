package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/response"
)

const BankName = "Mornee"

type WalletMiniData struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	BankName      string `json:"bank_name"`
}
type WalletResponseData struct {
	ID            string    `json:"id"`
	AccountNumber string    `json:"account_number"`
	BankName      string    `json:"bank_name"`
	Balance       float64   `json:"balance"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *RouteHandler) generateWallet(user_id string, phone_number string, tx *sql.Tx) (*database.Wallet, error) {

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

	_, err := h.DB.CreateWallet(userWallet, tx)
	if err != nil {
		return nil, err
	}
	return userWallet, nil

}

func (h *RouteHandler) HandleWalletBalance(w http.ResponseWriter, r *http.Request) {
	user := context.ContextGetAuthenticatedUser((r))

	walletID := r.PathValue("id")

	wallet, err := h.DB.GetWalletBalance(walletID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// check if logged in user is the owner of the wallet
	if user.ID != wallet.UserID {
		message := "Access denied"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		return
	}

	message := "Balance fetched successfully"

	data := map[string]any{
		"balance":  wallet.Balance,
		"currency": wallet.Currency,
	}
	err = response.JSONOkResponse(w, data, message, nil)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleWalletDetails(w http.ResponseWriter, r *http.Request) {
	user := context.ContextGetAuthenticatedUser((r))

	walletID := r.PathValue("id")

	wallet, found, err := h.DB.GetWallet(walletID)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		response.JSONErrorResponse(w, nil, ErrWalletNotFound.Error(), http.StatusUnprocessableEntity, nil)
		return
	}

	// check if logged in user is the owner of the wallet
	if user.ID != wallet.UserID {
		message := "Access denied"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		return
	}

	message := "Wallet details fetched successfully"

	data := &WalletResponseData{
		ID:            wallet.ID,
		Balance:       wallet.Balance,
		BankName:      BankName,
		Currency:      wallet.Currency,
		AccountNumber: wallet.AccountNumber,
		Status:        wallet.Status,
		CreatedAt:     wallet.CreatedAt,
	}
	err = response.JSONOkResponse(w, data, message, nil)

	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *RouteHandler) HandleUserWallets(w http.ResponseWriter, r *http.Request) {
	user := context.ContextGetAuthenticatedUser(r)

	wallets, found, err := h.DB.GetWalletsByUserId(user.ID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		message := "No wallet found"
		err = response.JSONOkResponse(w, []WalletResponseData{}, message, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	message := "Wallet details retrieved successfully"

	data := make([]*WalletResponseData, len(wallets))
	for i, wallet := range wallets {
		data[i] = &WalletResponseData{
			ID:            wallet.ID,
			Balance:       wallet.Balance,
			BankName:      BankName,
			Currency:      wallet.Currency,
			AccountNumber: wallet.AccountNumber,
			Status:        wallet.Status,
			CreatedAt:     wallet.CreatedAt,
		}
	}

	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}
