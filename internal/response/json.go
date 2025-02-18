package response

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

type Response[T any] struct {
	Status  int    `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	Error   T      `json:"error,omitempty"`
}

func JSONCreatedResponse(w http.ResponseWriter, data any, message string) error {
	if message == "" {
		message = "Request successful"
	}

	convertedData, ok := data.(map[string]any)
	if ok {
		data = ConvertKeysToSnakeCase(convertedData)
	}

	response := &Response[any]{
		Status:  http.StatusCreated,
		Success: true,
		Message: message,
		Data:    data,
	}

	return JSONWithHeaders(w, response, nil)
}

func JSONOkResponse(w http.ResponseWriter, data any, message string, headers http.Header) error {
	if message == "" {
		message = "Request successful"
	}

	convertedData, ok := data.(map[string]any)

	if ok {
		data = ConvertKeysToSnakeCase(convertedData)
	}

	response := &Response[any]{
		Status:  http.StatusOK,
		Success: true,
		Message: message,
		Data:    data,
	}

	return JSONWithHeaders(w, response, headers)
}

func JSONErrorResponse(w http.ResponseWriter, err any, message string, status int, headers http.Header) error {
	if message == "" {
		message = "Request failed"
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	response := &Response[any]{
		Status:  status,
		Success: false,
		Message: message,
		Error:   err,
	}

	return JSONWithHeaders(w, response, headers)
}

func JSON[T any](w http.ResponseWriter, response *Response[T]) error {
	return JSONWithHeaders(w, response, nil)
}

func JSONWithHeaders[T any](w http.ResponseWriter, response *Response[T], headers http.Header) error {

	js, err := json.MarshalIndent(response, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)

	w.Write(js)

	return nil
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
