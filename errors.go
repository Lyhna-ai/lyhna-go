package lyhna

import "fmt"

// LyhnaError is the base error type for all Lyhna API errors.
type LyhnaError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *LyhnaError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("lyhna: %s (HTTP %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("lyhna: %s", e.Message)
}

// AuthError is returned on 401 or 403 responses.
type AuthError struct {
	LyhnaError
}

// TimeoutError is returned when a request exceeds the configured timeout.
type TimeoutError struct {
	LyhnaError
}
