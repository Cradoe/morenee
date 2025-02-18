package helper

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type HelperRepository struct {
	baseUrl *string
	WG      *sync.WaitGroup
}

func New(baseUrl *string, wg *sync.WaitGroup) *HelperRepository {
	return &HelperRepository{
		baseUrl: baseUrl,
		WG:      wg,
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

func toSnakeCase(s string) string {
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

func ConvertKeysToSnakeCase(data map[string]interface{}) map[string]interface{} {
	snakeData := make(map[string]interface{})

	for key, value := range data {
		snakeKey := toSnakeCase(key)

		// Recursively handle nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			value = ConvertKeysToSnakeCase(nestedMap)
		}

		snakeData[snakeKey] = value
	}
	return snakeData
}
