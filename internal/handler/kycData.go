package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/context"
	"github.com/cradoe/morenee/internal/errHandler"
	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/request"
	"github.com/cradoe/morenee/internal/response"
	"github.com/cradoe/morenee/internal/validator"
)

type UserKYCDataResponse struct {
	ID          string                     `json:"id"`
	Value       string                     `json:"value"`
	Verified    bool                       `json:"verified"`
	CreatedAt   time.Time                  `json:"created_at"`
	Requirement KYCRequirementResponseData `json:"requirement"`
}

type UserKycDataHandler struct {
	UserKycDataRepo    repository.UserKycDataRepository
	KycRequirementRepo repository.KycRequirementRepository

	ErrHandler *errHandler.ErrorHandler
	Helper     *helper.HelperRepository
}

func NewUserKycDataHandler(handler *UserKycDataHandler) *UserKycDataHandler {
	return &UserKycDataHandler{
		UserKycDataRepo:    handler.UserKycDataRepo,
		KycRequirementRepo: handler.KycRequirementRepo,
		ErrHandler:         handler.ErrHandler,
		Helper:             handler.Helper,
	}
}

func (h *UserKycDataHandler) HandleSaveUserBVN(w http.ResponseWriter, r *http.Request) {
	var input struct {
		BVN       string              `json:"bvn"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.BVN), "Value is required")
	input.Validator.Check(len(input.BVN) == 10, "BVN should be 10 digits")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	user := context.ContextGetAuthenticatedUser((r))

	requirement, found, err := h.KycRequirementRepo.FindByName("BVN")
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// check that record has not been set
	_, found, err = h.UserKycDataRepo.GetByRequirementId(user.ID, requirement.ID)
	if found {
		message := "Data has already been set"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	err = h.UserKycDataRepo.Insert(user.ID, input.BVN, requirement.ID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// make attempt to upgrade user kyc level in background
	h.Helper.BackgroundTask(r, func() error {
		_, err = h.UserKycDataRepo.UpgradeLevel(user.ID)

		if err != nil {
			log.Printf("Error upgrading user kyc: %v", err)
			return err
		}

		return nil
	})

	message := "BVN saved successfully."
	err = response.JSONCreatedResponse(w, nil, message)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

func (h *UserKycDataHandler) HandleGetAllUserKYCData(w http.ResponseWriter, r *http.Request) {

	user := context.ContextGetAuthenticatedUser((r))

	kycDataList, err := h.UserKycDataRepo.GetAll(user.ID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	formattedResponse := make([]UserKYCDataResponse, len(kycDataList))
	for i, data := range kycDataList {
		formattedResponse[i] = UserKYCDataResponse{
			ID:        data.ID,
			Value:     data.SubmissionData,
			Verified:  data.Verified,
			CreatedAt: data.CreatedAt,
			Requirement: KYCRequirementResponseData{
				ID:          data.RequirementID,
				Requirement: data.Requirement,
			},
		}
	}

	message := "KYC data retrieved successfully."
	err = response.JSONOkResponse(w, formattedResponse, message, nil)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}

// general purpse handler for setting kyc data
func (h *UserKycDataHandler) HandleSaveKYCData(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RequirementID string              `json:"requirement_id"`
		Value         string              `json:"value"`
		Validator     validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		h.ErrHandler.BadRequest(w, r, err)
		return
	}

	input.Validator.Check(validator.NotBlank(input.RequirementID), "Requirement ID is required")
	input.Validator.Check(validator.NotBlank(input.Value), "Value is required")

	if input.Validator.HasErrors() {
		h.ErrHandler.FailedValidation(w, r, input.Validator.Errors)
		return
	}

	user := context.ContextGetAuthenticatedUser((r))

	// check that record has not been set
	_, found, err := h.UserKycDataRepo.GetByRequirementId(user.ID, input.RequirementID)
	if found {
		message := "Data has already been set"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	err = h.UserKycDataRepo.Insert(user.ID, input.Value, input.RequirementID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// make attempt to upgrade user kyc level in background
	h.Helper.BackgroundTask(r, func() error {
		_, err = h.UserKycDataRepo.UpgradeLevel(user.ID)

		if err != nil {
			log.Printf("Error upgrading user kyc: %v", err)
			return err
		}

		return nil
	})

	message := "KYC data saved successfully."
	err = response.JSONCreatedResponse(w, nil, message)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
	}
}
