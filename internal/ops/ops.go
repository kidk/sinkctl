package ops

import (
	"context"
	"net/http"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/config"
	"github.com/kidk/sinkctl/internal/version"
)

// Ops runs operations against one Bugsink instance.
type Ops struct {
	api *bugsink.ClientWithResponses //nolint:unused // used by the first API op
}

func New(c config.Context, hc *http.Client) (*Ops, error) {
	api, err := NewClient(c, hc)
	if err != nil {
		return nil, err
	}
	return &Ops{api: api}, nil
}

// NewClient builds the authenticated API client; exported for e2e tests.
func NewClient(c config.Context, hc *http.Client) (*bugsink.ClientWithResponses, error) {
	return bugsink.NewClientWithResponses(c.URL,
		bugsink.WithHTTPClient(hc),
		bugsink.WithRequestEditorFn(func(_ context.Context, r *http.Request) error {
			r.Header.Set("Authorization", "Bearer "+c.Token)
			r.Header.Set("User-Agent", "sinkctl/"+version.Version)
			return nil
		}))
}
