package mocks

import (
	"database/sql"

	"github.com/cradoe/morenee/internal/models"
	"github.com/stretchr/testify/mock"
)

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
