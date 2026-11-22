package ops

import (
	"fmt"
	"net/http"
	"strings"
)

// Kind classifies errors for agents: CLI maps it to an exit code, MCP puts it in the tool error.
type Kind string

const (
	KindError    Kind = "error"
	KindUsage    Kind = "usage"
	KindAuth     Kind = "auth"
	KindNotFound Kind = "not_found"
	KindAPI      Kind = "api"
	KindNetwork  Kind = "network"
)

type Error struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// HTTPError maps a non-2xx API response to an *Error; nil for 2xx.
func HTTPError(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	kind := KindAPI
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		kind = KindAuth
	case http.StatusNotFound:
		kind = KindNotFound
	}
	msg := fmt.Sprintf("%d %s", status, http.StatusText(status))
	if b := strings.TrimSpace(string(body)); b != "" {
		if len(b) > 500 {
			b = b[:500] + "..."
		}
		msg += ": " + b
	}
	return &Error{Kind: kind, Message: msg, Status: status}
}
