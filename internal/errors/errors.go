// Package errors provides shared error types that map to both CLI exit codes
// and HTTP status codes, enabling consistent error handling across the CLI and API.
package errors

import (
	"fmt"
	"net/http"
)

// Kind represents the category of an error, which determines both the
// CLI exit code and HTTP status code.
type Kind int

const (
	// KindInvalidArgs represents invalid input arguments.
	// CLI exit code: 2, HTTP status: 400 Bad Request
	KindInvalidArgs Kind = iota

	// KindNotFound represents a missing resource.
	// CLI exit code: 3, HTTP status: 404 Not Found
	KindNotFound

	// KindStateError represents an invalid state transition.
	// CLI exit code: 4, HTTP status: 422 Unprocessable Entity
	KindStateError

	// KindConcurrentConflict represents a concurrent modification conflict.
	// CLI exit code: 6, HTTP status: 409 Conflict
	KindConcurrentConflict

	// KindInternal represents an internal/database error.
	// CLI exit code: 5, HTTP status: 500 Internal Server Error
	KindInternal

	// KindGeneral represents a general error that doesn't fit other categories.
	// CLI exit code: 1, HTTP status: 500 Internal Server Error
	KindGeneral
)

// String returns a human-readable name for the error kind.
func (k Kind) String() string {
	switch k {
	case KindInvalidArgs:
		return "InvalidArgs"
	case KindNotFound:
		return "NotFound"
	case KindStateError:
		return "StateError"
	case KindConcurrentConflict:
		return "ConcurrentConflict"
	case KindInternal:
		return "Internal"
	case KindGeneral:
		return "General"
	default:
		return "Unknown"
	}
}

// Error represents a structured error with kind, message, cause, and optional details.
// It implements the standard error interface and provides methods for mapping to
// CLI exit codes and HTTP status codes.
type Error struct {
	Kind       Kind
	Message    string
	Cause      error
	Details    map[string]interface{}
	Suggestion string // Optional suggestion for resolving the error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause, enabling errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Cause
}

// CLIExitCode returns the appropriate CLI exit code for this error.
func (e *Error) CLIExitCode() int {
	switch e.Kind {
	case KindInvalidArgs:
		return 2
	case KindNotFound:
		return 3
	case KindStateError:
		return 4
	case KindInternal:
		return 5
	case KindConcurrentConflict:
		return 6
	case KindGeneral:
		return 1
	default:
		return 1
	}
}

// HTTPStatus returns the appropriate HTTP status code for this error.
func (e *Error) HTTPStatus() int {
	switch e.Kind {
	case KindInvalidArgs:
		return http.StatusBadRequest // 400
	case KindNotFound:
		return http.StatusNotFound // 404
	case KindStateError:
		return http.StatusUnprocessableEntity // 422
	case KindConcurrentConflict:
		return http.StatusConflict // 409
	case KindInternal:
		return http.StatusInternalServerError // 500
	case KindGeneral:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError
	}
}

// WithDetails adds details to the error and returns it for chaining.
func (e *Error) WithDetails(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithSuggestion adds a suggestion to the error and returns it for chaining.
func (e *Error) WithSuggestion(suggestion string) *Error {
	e.Suggestion = suggestion
	return e
}

// Constructor functions

// NotFound creates an error for missing resources.
func NotFound(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

// InvalidArgs creates an error for invalid arguments.
func InvalidArgs(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindInvalidArgs,
		Message: fmt.Sprintf(format, args...),
	}
}

// StateError creates an error for invalid state transitions.
func StateError(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindStateError,
		Message: fmt.Sprintf(format, args...),
	}
}

// ConcurrentConflict creates an error for concurrent modification conflicts.
func ConcurrentConflict(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindConcurrentConflict,
		Message: fmt.Sprintf(format, args...),
	}
}

// Internal creates an error for internal/database errors.
func Internal(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindInternal,
		Message: fmt.Sprintf(format, args...),
	}
}

// General creates a general error.
func General(format string, args ...interface{}) *Error {
	return &Error{
		Kind:    KindGeneral,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with a specific kind and message.
func Wrap(err error, kind Kind, format string, args ...interface{}) *Error {
	return &Error{
		Kind:    kind,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WrapInternal wraps an error as an internal error.
func WrapInternal(err error, format string, args ...interface{}) *Error {
	return Wrap(err, KindInternal, format, args...)
}

// Helper functions for extracting error information

// GetKind extracts the Kind from an error, returning KindGeneral if the error
// is not an *Error.
func GetKind(err error) Kind {
	if e, ok := err.(*Error); ok {
		return e.Kind
	}
	return KindGeneral
}

// GetCLIExitCode extracts the CLI exit code from an error.
func GetCLIExitCode(err error) int {
	if e, ok := err.(*Error); ok {
		return e.CLIExitCode()
	}
	return 1 // General error
}

// GetHTTPStatus extracts the HTTP status code from an error.
func GetHTTPStatus(err error) int {
	if e, ok := err.(*Error); ok {
		return e.HTTPStatus()
	}
	return http.StatusInternalServerError
}

// Is returns true if the error is of the specified kind.
func Is(err error, kind Kind) bool {
	if e, ok := err.(*Error); ok {
		return e.Kind == kind
	}
	return false
}
