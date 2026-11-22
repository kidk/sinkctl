package ops

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// transportErr classifies a generated-client error; cancellation passes through untouched.
func transportErr(what string, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return &Error{Kind: KindNetwork, Message: fmt.Sprintf("%s: %v", what, err)}
}

// statusErr is HTTPError with what prefixed to the message.
func statusErr(what string, res *http.Response, body []byte) error {
	err := HTTPError(res.StatusCode, body)
	var e *Error
	if errors.As(err, &e) {
		e.Message = what + ": " + e.Message
	}
	return err
}

// decoded checks status, then returns the decoded 200 body.
func decoded[T any](what string, res *http.Response, body []byte, v *T) (T, error) {
	var zero T
	if err := statusErr(what, res, body); err != nil {
		return zero, err
	}
	if v == nil {
		return zero, &Error{
			Kind:    KindAPI,
			Message: fmt.Sprintf("%s: unexpected response body (%s): %s", what, res.Header.Get("Content-Type"), snippet(body)),
			Status:  res.StatusCode,
		}
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

// optEnum is ptr for a generated enum, rejecting values the API would 400 on.
func optEnum[T interface {
	~string
	Valid() bool
}](field, v, allowed string) (*T, error) {
	if v == "" {
		return nil, nil
	}
	e := T(v)
	if !e.Valid() {
		return nil, usagef("%s: invalid value %q; want %s", field, v, allowed)
	}
	return &e, nil
}
