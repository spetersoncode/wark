// Package common provides shared utilities used across CLI and server packages.
package common

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalidTicketKey is returned when a ticket key doesn't match the expected format.
var ErrInvalidTicketKey = errors.New("invalid ticket key format (expected PROJECT-NUMBER)")

// ticketKeyRegex validates ticket keys like "WEBAPP-42" or "TEST123-1"
var ticketKeyRegex = regexp.MustCompile(`^([A-Z][A-Z0-9]*)-(\d+)$`)

// ParseTicketKey parses a ticket key like "WEBAPP-42" into project key and number.
// It also accepts just a number (e.g., "42") for use with a default project.
// Returns ErrInvalidTicketKey if the format is invalid or number is not positive.
func ParseTicketKey(key string) (projectKey string, number int, err error) {
	key = strings.ToUpper(strings.TrimSpace(key))

	// Pattern: PROJECT-NUMBER (e.g., "WEBAPP-42")
	matches := ticketKeyRegex.FindStringSubmatch(key)
	if matches != nil {
		projectKey = matches[1]
		number, _ = strconv.Atoi(matches[2])
		return projectKey, number, nil
	}

	// Just a positive number (e.g., "42")
	if n, err := strconv.Atoi(key); err == nil && n > 0 {
		return "", n, nil
	}

	return "", 0, ErrInvalidTicketKey
}
