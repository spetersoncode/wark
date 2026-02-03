package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		kind     Kind
		expected string
	}{
		{KindInvalidArgs, "InvalidArgs"},
		{KindNotFound, "NotFound"},
		{KindStateError, "StateError"},
		{KindConcurrentConflict, "ConcurrentConflict"},
		{KindInternal, "Internal"},
		{KindGeneral, "General"},
		{Kind(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.expected {
				t.Errorf("Kind.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorImplementsError(t *testing.T) {
	err := NotFound("ticket %s not found", "PROJ-123")

	var _ error = err // Compile-time check that *Error implements error

	if err.Error() != "ticket PROJ-123 not found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "ticket PROJ-123 not found")
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := errors.New("database connection failed")
	err := WrapInternal(cause, "failed to fetch ticket")

	expected := "failed to fetch ticket: database connection failed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(cause, KindInternal, "wrapped error")

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is compatibility
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

func TestCLIExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected int
	}{
		{"InvalidArgs", InvalidArgs("bad input"), 2},
		{"NotFound", NotFound("not found"), 3},
		{"StateError", StateError("invalid state"), 4},
		{"Internal", Internal("db error"), 5},
		{"ConcurrentConflict", ConcurrentConflict("conflict"), 6},
		{"General", General("general error"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.CLIExitCode(); got != tt.expected {
				t.Errorf("CLIExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected int
	}{
		{"InvalidArgs", InvalidArgs("bad input"), http.StatusBadRequest},
		{"NotFound", NotFound("not found"), http.StatusNotFound},
		{"StateError", StateError("invalid state"), http.StatusUnprocessableEntity},
		{"ConcurrentConflict", ConcurrentConflict("conflict"), http.StatusConflict},
		{"Internal", Internal("db error"), http.StatusInternalServerError},
		{"General", General("general error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HTTPStatus(); got != tt.expected {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		kind     Kind
		message  string
	}{
		{
			name:    "NotFound",
			err:     NotFound("ticket %s not found", "PROJ-1"),
			kind:    KindNotFound,
			message: "ticket PROJ-1 not found",
		},
		{
			name:    "InvalidArgs",
			err:     InvalidArgs("invalid status: %s", "unknown"),
			kind:    KindInvalidArgs,
			message: "invalid status: unknown",
		},
		{
			name:    "StateError",
			err:     StateError("cannot transition from %s to %s", "done", "ready"),
			kind:    KindStateError,
			message: "cannot transition from done to ready",
		},
		{
			name:    "ConcurrentConflict",
			err:     ConcurrentConflict("ticket already claimed by %s", "worker-1"),
			kind:    KindConcurrentConflict,
			message: "ticket already claimed by worker-1",
		},
		{
			name:    "Internal",
			err:     Internal("database error"),
			kind:    KindInternal,
			message: "database error",
		},
		{
			name:    "General",
			err:     General("something went wrong"),
			kind:    KindGeneral,
			message: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", tt.err.Kind, tt.kind)
			}
			if tt.err.Message != tt.message {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.message)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := Wrap(cause, KindInternal, "failed to connect to database")

	if err.Kind != KindInternal {
		t.Errorf("Kind = %v, want %v", err.Kind, KindInternal)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Message != "failed to connect to database" {
		t.Errorf("Message = %q, want %q", err.Message, "failed to connect to database")
	}
}

func TestWithDetails(t *testing.T) {
	err := NotFound("ticket not found").
		WithDetails("ticket_key", "PROJ-123").
		WithDetails("project", "PROJ")

	if err.Details == nil {
		t.Fatal("Details is nil")
	}
	if err.Details["ticket_key"] != "PROJ-123" {
		t.Errorf("Details[ticket_key] = %v, want %q", err.Details["ticket_key"], "PROJ-123")
	}
	if err.Details["project"] != "PROJ" {
		t.Errorf("Details[project] = %v, want %q", err.Details["project"], "PROJ")
	}
}

func TestWithSuggestion(t *testing.T) {
	err := NotFound("ticket not found").
		WithSuggestion("Run 'wark ticket list' to see available tickets")

	if err.Suggestion != "Run 'wark ticket list' to see available tickets" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "Run 'wark ticket list' to see available tickets")
	}
}

func TestGetKind(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Kind
	}{
		{"NotFound error", NotFound("not found"), KindNotFound},
		{"InvalidArgs error", InvalidArgs("bad input"), KindInvalidArgs},
		{"Standard error", errors.New("standard error"), KindGeneral},
		{"Nil wrapped", Wrap(nil, KindStateError, "state error"), KindStateError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetKind(tt.err); got != tt.expected {
				t.Errorf("GetKind() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetCLIExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"NotFound error", NotFound("not found"), 3},
		{"Standard error", errors.New("standard error"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCLIExitCode(tt.err); got != tt.expected {
				t.Errorf("GetCLIExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"NotFound error", NotFound("not found"), http.StatusNotFound},
		{"Standard error", errors.New("standard error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatus(tt.err); got != tt.expected {
				t.Errorf("GetHTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		kind     Kind
		expected bool
	}{
		{"matching kind", NotFound("not found"), KindNotFound, true},
		{"non-matching kind", NotFound("not found"), KindInvalidArgs, false},
		{"standard error", errors.New("standard"), KindNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.kind); got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChaining(t *testing.T) {
	// Test that WithDetails and WithSuggestion can be chained
	err := NotFound("ticket %s not found", "PROJ-123").
		WithDetails("ticket_key", "PROJ-123").
		WithDetails("searched_at", "2024-01-01").
		WithSuggestion("Check the ticket key format")

	if err.Kind != KindNotFound {
		t.Errorf("Kind = %v, want %v", err.Kind, KindNotFound)
	}
	if len(err.Details) != 2 {
		t.Errorf("len(Details) = %d, want 2", len(err.Details))
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}
