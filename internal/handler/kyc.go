package handler

import (
	"net/http"

	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/response"
)

type KYCResponseData struct {
	ID                  string                       `json:"id"`
	LevelName           string                       `json:"level_name"`
	DailyTransferLimit  float64                      `json:"daily_transfer_limit"`
	WalletBalanceLimit  float64                      `json:"wallet_balance_limit"`
	SingleTransferLimit float64                      `json:"single_transfer_limit"`
	Requirements        []KYCRequirementResponseData `json:"requirements"`
}

type KYCRequirementResponseData struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement"`
}

type KycHandler struct {
	KycRepo repository.KycRepository

	ErrHandler *errHandler.ErrorHandler
}

func NewKycHandler(handler *KycHandler) *KycHandler {
	return &KycHandler{
		KycRepo:    handler.KycRepo,
		ErrHandler: handler.ErrHandler,
	}
}

func (h *KycHandler) HandleKYCs(w http.ResponseWriter, r *http.Request) {

	KYCS, err := h.KycRepo.GetAll()
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if len(KYCS) == 0 {
		message := "No KYC found"
		err = response.JSONOkResponse(w, []KYCResponseData{}, message, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	message := "Data retrieved successfully"

	data := make([]*KYCResponseData, len(KYCS))
	for i, kyc := range KYCS {
		requirements := make([]KYCRequirementResponseData, len(kyc.Requirements))
		for j, req := range kyc.Requirements {
			requirements[j] = KYCRequirementResponseData{
				ID:          req.ID,
				Requirement: req.Requirement,
			}
		}

		data[i] = &KYCResponseData{
			ID:                  kyc.ID,
			LevelName:           kyc.LevelName,
			DailyTransferLimit:  kyc.DailyTransferLimit,
			WalletBalanceLimit:  kyc.WalletBalanceLimit,
			SingleTransferLimit: kyc.SingleTransferLimit,
			Requirements:        requirements,
		}
	}

	err = response.JSONOkResponse(w, data, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *KycHandler) HandleSingleYC(w http.ResponseWriter, r *http.Request) {
	kycID := r.PathValue("id")

	result, found, err := h.KycRepo.GetOne(kycID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		h.ErrHandler.NotFound(w, r)
		return
	}

	message := "Data retrieved successfully"

	kyc := &KYCResponseData{
		ID:                  result.ID,
		LevelName:           result.LevelName,
		DailyTransferLimit:  result.DailyTransferLimit,
		SingleTransferLimit: result.SingleTransferLimit,
		WalletBalanceLimit:  result.WalletBalanceLimit,
	}

	requirements := make([]KYCRequirementResponseData, len(result.Requirements))
	for j, req := range result.Requirements {
		requirements[j] = KYCRequirementResponseData{
			ID:          req.ID,
			Requirement: req.Requirement,
		}
	}

	kyc.Requirements = requirements

	err = response.JSONOkResponse(w, kyc, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}
