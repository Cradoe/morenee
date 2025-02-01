package helper

import (
	"net/http"
	"sync"
)

type HelperRepository struct {
	baseUrl *string
	WG      *sync.WaitGroup
}

func NewHelperRepository(baseUrl *string) *HelperRepository {
	return &HelperRepository{
		baseUrl: baseUrl,
	}
}

func (h *HelperRepository) NewEmailData() map[string]any {
	data := map[string]any{
		"BaseURL": h.baseUrl,
	}

	return data
}

func (h *HelperRepository) BackgroundTask(r *http.Request, fn func() error) {
	h.WG.Add(1)

	go func() {
		defer h.WG.Done()

		defer func() {
			err := recover()
			if err != nil {
				// h.errorHandler.ReportServerError(r, fmt.Errorf("%s", err))
			}
		}()

		err := fn()
		if err != nil {
			// h.errorHandler.ReportServerError(r, err)
		}
	}()
}
