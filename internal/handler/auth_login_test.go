package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cradoe/morenee/internal/config"
	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/models"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepo implements UserRepository but only mocks the needed methods.
type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) CheckIfPhoneNumberExist(phoneNumber string) (bool, error) {
	return false, nil
}

func (m *MockUserRepo) Insert(user *models.User, tx *sql.Tx) (string, error) {
	return "", nil
}

func (m *MockUserRepo) GetOne(id string) (*models.User, bool, error) {
	return nil, false, nil
}

func (m *MockUserRepo) GetByEmail(email string) (*models.User, bool, error) {
	args := m.Called(email)
	return args.Get(0).(*models.User), args.Bool(1), args.Error(2)
}

func (m *MockUserRepo) Verify(id string, tx *sql.Tx) error {
	return nil
}

func (m *MockUserRepo) UpdatePassword(id, password string) error {
	return nil
}

func (m *MockUserRepo) ChangePin(id, pin string) error {
	return nil
}

func (m *MockUserRepo) ChangeProfilePicture(id, image string) error {
	return nil
}

func (m *MockUserRepo) Lock(id string) error {
	return nil
}

type MockActivityRepo struct {
	mock.Mock
}

func (m *MockActivityRepo) CountConsecutiveFailedLoginAttempts(userID, action_desc string) int {
	return 0
}

func (m *MockActivityRepo) Insert(log *repository.ActivityLog) (*repository.ActivityLog, error) {
	args := m.Called(log)
	return args.Get(0).(*repository.ActivityLog), args.Error(1)
}

type MockHelper struct{}

func (m *MockHelper) BackgroundTask(r *http.Request, fn func() error) {
	go func() {
		err := fn()
		if err != nil {
			log.Printf("Background task error: %v", err)
		}
	}()
}

// MockErrorHandler simulates error handling inside HelperRepository.
type MockErrorHandler struct{}

func (m *MockErrorHandler) ReportServerError(r *http.Request, err error) {
	log.Printf("Mock Error Handler: %v", err)
}

type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) Send(recipient string, data any, patterns ...string) error {
	args := m.Called(recipient, data, patterns)
	return args.Error(0)
}

func TestHandleAuthLogin_ValidCredentials(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepo)
	mockActivityRepo := new(MockActivityRepo)
	mockMailer := new(MockMailer)

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

	// ✅ Add mock Config
	mockConfig := &config.Config{
		BaseURL:  "http://localhost",
		HttpPort: 8080,
		Db: struct {
			Dsn         string
			Automigrate bool
		}{
			Dsn:         "mock_dsn",
			Automigrate: false,
		},
		RedisServer: "localhost:6379",
		Jwt: struct {
			SecretKey string
		}{
			SecretKey: "test_secret",
		},
		Notifications: struct {
			Email string
		}{
			Email: "no-reply@example.com",
		},
		Smtp: struct {
			Host     string
			Port     int
			Username string
			Password string
			From     string
		}{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user@example.com",
			Password: "password",
			From:     "no-reply@example.com",
		},
		KafkaServers: "localhost:9092",
	}

	authHandler := &AuthHandler{
		UserRepo:     mockUserRepo,
		ActivityRepo: mockActivityRepo,
		Helper:       mockHelper,
		Mailer:       mockMailer,
		Config:       mockConfig, // ✅ Include Config
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
