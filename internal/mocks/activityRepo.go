package mocks

import (
	"github.com/cradoe/morenee/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockActivityRepo struct {
	mock.Mock
}

func (m *MockActivityRepo) CountConsecutiveFailedLoginAttempts(userID, action_desc string) int {
	return 0
}

func (m *MockActivityRepo) Insert(log *models.ActivityLog) (*models.ActivityLog, error) {
	args := m.Called(log)
	return args.Get(0).(*models.ActivityLog), args.Error(1)
}
