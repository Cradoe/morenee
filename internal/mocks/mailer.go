package mocks

import "github.com/stretchr/testify/mock"

type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) Send(recipient string, data any, patterns ...string) error {
	args := m.Called(recipient, data, patterns)
	return args.Error(0)
}
