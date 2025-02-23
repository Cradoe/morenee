package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/cradoe/morenee/internal/context"
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

func (h *RouteHandler) HandleSaveUserBVN(w http.ResponseWriter, r *http.Request) {
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

	requirement, found, err := h.DB.KycRequirement().FindByName("BVN")
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	if !found {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// check that record has not been set
	_, found, err = h.DB.UserKycData().GetByRequirementId(user.ID, requirement.ID)
	if found {
		message := "Data has already been set"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	err = h.DB.UserKycData().Insert(user.ID, input.BVN, requirement.ID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// make attempt to upgrade user kyc level in background
	h.Helper.BackgroundTask(r, func() error {
		_, err = h.DB.UserKycData().UpgradeLevel(user.ID)

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

func (h *RouteHandler) HandleGetAllUserKYCData(w http.ResponseWriter, r *http.Request) {

	user := context.ContextGetAuthenticatedUser((r))

	kycDataList, err := h.DB.UserKycData().GetAll(user.ID)
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
func (h *RouteHandler) HandleSaveKYCData(w http.ResponseWriter, r *http.Request) {
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
	_, found, err := h.DB.UserKycData().GetByRequirementId(user.ID, input.RequirementID)
	if found {
		message := "Data has already been set"
		response.JSONErrorResponse(w, nil, message, http.StatusForbidden, nil)
		if err != nil {
			h.ErrHandler.ServerError(w, r, err)
		}
		return
	}

	err = h.DB.UserKycData().Insert(user.ID, input.Value, input.RequirementID)
	if err != nil {
		h.ErrHandler.ServerError(w, r, err)
		return
	}

	// make attempt to upgrade user kyc level in background
	h.Helper.BackgroundTask(r, func() error {
		_, err = h.DB.UserKycData().UpgradeLevel(user.ID)

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
