package cli

import (
	"errors"
	"fmt"
	"strings"

	werrors "github.com/spetersoncode/wark/internal/errors"
)

// WarkError is an error with an exit code and optional suggestion.
// It wraps the shared errors.Error type to maintain backward compatibility
// while leveraging the shared error infrastructure.
type WarkError struct {
	Code       int
	Message    string
	Cause      error
	Suggestion string
}

func (e *WarkError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}
	return b.String()
}

func (e *WarkError) Unwrap() error {
	return e.Cause
}

// FormatError returns the error message with suggestion if present
func (e *WarkError) FormatError() string {
	var b strings.Builder
	b.WriteString("Error: ")
	b.WriteString(e.Error())
	if e.Suggestion != "" {
		b.WriteString("\n\nSuggestion: ")
		b.WriteString(e.Suggestion)
	}
	return b.String()
}

// ExitCode returns the exit code for any error.
// Supports both WarkError and the shared errors.Error type.
func ExitCode(err error) int {
	// Check for shared error type first
	var sharedErr *werrors.Error
	if errors.As(err, &sharedErr) {
		return sharedErr.CLIExitCode()
	}

	// Fall back to WarkError
	var werr *WarkError
	if errors.As(err, &werr) {
		return werr.Code
	}
	return ExitGeneralError
}

// FormatErrorMessage returns formatted error with suggestion if available.
// Supports both WarkError and the shared errors.Error type.
func FormatErrorMessage(err error) string {
	// Check for shared error type first
	var sharedErr *werrors.Error
	if errors.As(err, &sharedErr) {
		var b strings.Builder
		b.WriteString("Error: ")
		b.WriteString(sharedErr.Error())
		if sharedErr.Suggestion != "" {
			b.WriteString("\n\nSuggestion: ")
			b.WriteString(sharedErr.Suggestion)
		}
		return b.String()
	}

	// Fall back to WarkError
	var werr *WarkError
	if errors.As(err, &werr) {
		return werr.FormatError()
	}
	return "Error: " + err.Error()
}

// Error constructors with proper exit codes

// ErrInvalidArgs creates an error for invalid arguments (exit code 2)
func ErrInvalidArgs(format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitInvalidArgs,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrInvalidArgsWithSuggestion creates an error for invalid arguments with a suggestion
func ErrInvalidArgsWithSuggestion(suggestion, format string, args ...interface{}) error {
	return &WarkError{
		Code:       ExitInvalidArgs,
		Message:    fmt.Sprintf(format, args...),
		Suggestion: suggestion,
	}
}

// ErrNotFound creates an error for missing resources (exit code 3)
func ErrNotFound(format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrNotFoundWithSuggestion creates a not found error with a suggestion
func ErrNotFoundWithSuggestion(suggestion, format string, args ...interface{}) error {
	return &WarkError{
		Code:       ExitNotFound,
		Message:    fmt.Sprintf(format, args...),
		Suggestion: suggestion,
	}
}

// ErrStateError creates an error for invalid state transitions (exit code 4)
func ErrStateError(format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitStateError,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrStateErrorWithSuggestion creates a state error with a suggestion
func ErrStateErrorWithSuggestion(suggestion, format string, args ...interface{}) error {
	return &WarkError{
		Code:       ExitStateError,
		Message:    fmt.Sprintf(format, args...),
		Suggestion: suggestion,
	}
}

// ErrDatabase creates an error for database operations (exit code 5)
func ErrDatabase(cause error, format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitDBError,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// ErrDatabaseWithSuggestion creates a database error with a suggestion
func ErrDatabaseWithSuggestion(cause error, suggestion, format string, args ...interface{}) error {
	return &WarkError{
		Code:       ExitDBError,
		Message:    fmt.Sprintf(format, args...),
		Cause:      cause,
		Suggestion: suggestion,
	}
}

// ErrConcurrentConflict creates an error for concurrent modification (exit code 6)
func ErrConcurrentConflict(format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitConcurrentConflict,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrConcurrentConflictWithSuggestion creates a conflict error with a suggestion
func ErrConcurrentConflictWithSuggestion(suggestion, format string, args ...interface{}) error {
	return &WarkError{
		Code:       ExitConcurrentConflict,
		Message:    fmt.Sprintf(format, args...),
		Suggestion: suggestion,
	}
}

// ErrGeneral creates a general error (exit code 1)
func ErrGeneral(format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitGeneralError,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrGeneralWithCause creates a general error with a cause
func ErrGeneralWithCause(cause error, format string, args ...interface{}) error {
	return &WarkError{
		Code:    ExitGeneralError,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// WrapWithCode wraps an existing error with a specific exit code
func WrapWithCode(code int, err error, format string, args ...interface{}) error {
	return &WarkError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// Common suggestions
const (
	SuggestRunInit        = "Run 'wark init' to create a new database."
	SuggestCheckTicketKey = "Check the ticket key format. It should be like PROJECT-123."
	SuggestListProjects   = "Run 'wark project list' to see available projects."
	SuggestListTickets    = "Run 'wark ticket list' to see available tickets."
	SuggestCheckStatus    = "Run 'wark ticket show %s' to check the ticket's current status."
	SuggestReleaseClaim   = "The ticket may be claimed by another worker. Run 'wark claim list --active' to see."
	SuggestWaitOrRetry    = "Wait for the current operation to complete, or try again."
)
