package handler

import (
	"database/sql"

	"github.com/cradoe/morenee/internal/database"
)

type walletHandler struct {
	db *database.DB
}

func NewWalletHandler(db *database.DB) *walletHandler {
	return &walletHandler{
		db: db,
	}
}

func (h *walletHandler) generateWallet(user_id int, tx *sql.Tx) (bool, error) {

	userWallet := &database.Wallet{
		UserID: user_id,
	}

	_, err := h.db.CreateWallet(userWallet, tx)
	if err != nil {
		return false, err
	}
	return true, nil

}
