package mocks

import (
	"github.com/cradoe/morenee/internal/repository"
	"github.com/stretchr/testify/mock"
)

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
