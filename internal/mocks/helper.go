package mocks

import (
	"log"
	"net/http"
)

type MockHelper struct{}

func (m *MockHelper) BackgroundTask(r *http.Request, fn func() error) {
	go func() {
		err := fn()
		if err != nil {
			log.Printf("Background task error: %v", err)
		}
	}()
}
