package response

import (
	"net/http"
)

type APIError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string { return e.Message }
func (e *APIError) Unwrap() error { return e.Err }

func BadRequest(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "Bad Request",
		Err:        err,
	}
}

func PayloadTooLarge(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusRequestEntityTooLarge,
		Message:    "File too large",
		Err:        err,
	}
}

func UnsupportedMediaType(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusUnsupportedMediaType,
		Message:    "Unsupported file type",
		Err:        err,
	}
}

func CannotProcessFile(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Message:    "Failed to process file",
		Err:        err,
	}
}

func TooManyRequests(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusTooManyRequests,
		Message:    "Too many requests",
	}
}

func InternalServerError(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
		Err:        err,
	}
}
