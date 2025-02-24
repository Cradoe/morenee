package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/mocks"
	"github.com/cradoe/morenee/internal/models"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleAuthLogin_ValidCredentials(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepo)
	mockActivityRepo := new(mocks.MockActivityRepo)
	mockMailer := new(mocks.MockMailer)

	var baseURL string = "http://localhost"
	var wg sync.WaitGroup
	mockHelper := helper.New(&baseURL, &wg, nil)

	mockMailer.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	testUser := &models.User{
		ID:             "123",
		Email:          "test@example.com",
		HashedPassword: "$2a$10$oiIYEECpY/GRNs9Fi7Yh1.o4Dw2fTD26eu5z48KYgXkMuOiWlSvqG",
		Status:         repository.UserAccountActiveStatus,
	}

	mockUserRepo.On("GetByEmail", "test@example.com").Return(testUser, true, nil)
	mockActivityRepo.On("Insert", mock.Anything).Return(&repository.ActivityLog{}, nil)

	authHandler := &AuthHandler{
		UserRepo:     mockUserRepo,
		ActivityRepo: mockActivityRepo,
		Helper:       mockHelper,
		Mailer:       mockMailer,
		Config:       mocks.MockConfig,
	}

	requestBody, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "correctpassword",
	})

	req, err := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Act
	authHandler.HandleAuthLogin(rr, req)

	// Assert
	require.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	require.Contains(t, response, "data")

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "Expected response['data'] to be a map")

	require.Contains(t, data, "auth_token")
	require.Contains(t, data, "token_expiry")
	require.NotEmpty(t, data["auth_token"])

	mockUserRepo.AssertExpectations(t)
	mockActivityRepo.AssertExpectations(t)
	mockMailer.AssertExpectations(t)
}
