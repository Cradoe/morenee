package helper

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/cradoe/morenee/internal/errHandler"
)

type Helper struct {
	baseUrl    *string
	WG         *sync.WaitGroup
	errHandler *errHandler.ErrorHandler
}

func New(baseUrl *string, wg *sync.WaitGroup, errHandler *errHandler.ErrorHandler) *Helper {
	return &Helper{
		baseUrl:    baseUrl,
		WG:         wg,
		errHandler: errHandler,
	}
}

func (h *Helper) NewEmailData() map[string]any {
	data := map[string]any{
		"BaseURL": h.baseUrl,
	}

	return data
}

func (h *Helper) BackgroundTask(r *http.Request, fn func() error) {
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
