package helper

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/cradoe/morenee/internal/errHandler"
)

type HelperRepository struct {
	baseUrl    *string
	WG         *sync.WaitGroup
	errHandler *errHandler.ErrorHandler
}

func New(baseUrl *string, wg *sync.WaitGroup, errHandler *errHandler.ErrorHandler) *HelperRepository {
	return &HelperRepository{
		baseUrl:    baseUrl,
		WG:         wg,
		errHandler: errHandler,
	}
}

func (h *HelperRepository) NewEmailData() map[string]any {
	data := map[string]any{
		"BaseURL": h.baseUrl,
	}

	return data
}

func (h *HelperRepository) BackgroundTask(r *http.Request, fn func() error) {
	// h.WG.Add(1)

	go func() {
		// defer h.WG.Done()

		defer func() {
			err := recover()
			if err != nil {
				h.errHandler.ReportServerError(nil, fmt.Errorf("%s", err))
			}
		}()

		err := fn()
		if err != nil {
			h.errHandler.ReportServerError(nil, err)
		}
	}()
}
