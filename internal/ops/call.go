package ops

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// transportErr classifies a generated-client error; cancellation passes through untouched.
func transportErr(what string, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return &Error{Kind: KindNetwork, Message: fmt.Sprintf("%s: %v", what, err)}
}

// decoded checks status, then returns the decoded 200 body.
func decoded[T any](what string, status int, body []byte, v *T) (T, error) {
	var zero T
	if err := HTTPError(status, body); err != nil {
		var e *Error
		if errors.As(err, &e) {
			e.Message = what + ": " + e.Message
		}
		return zero, err
	}
	if v == nil {
		return zero, &Error{Kind: KindAPI, Message: fmt.Sprintf("%s: unexpected response body", what), Status: status}
	}
	return *v, nil
}

func usagef(format string, a ...any) error {
	return &Error{Kind: KindUsage, Message: fmt.Sprintf(format, a...)}
}

func parseUUID(field, s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, usagef("%s: invalid UUID %q", field, s)
	}
	return id, nil
}

// ptr returns nil for the zero value so optional query params are omitted.
func ptr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
