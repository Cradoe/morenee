package mocks

import (
	"log"
	"net/http"
)

type MockErrorHandler struct{}

func (m *MockErrorHandler) ReportServerError(r *http.Request, err error) {
	log.Printf("Mock Error Handler: %v", err)
}
