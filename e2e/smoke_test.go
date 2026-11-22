//go:build e2e

// Package e2e runs against a real Bugsink. Needs BUGSINK_URL and BUGSINK_TOKEN;
// `task e2e` starts a pinned container and sets both.
package e2e

import (
	"net/http"
	"os"
	"testing"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/config"
	"github.com/kidk/sinkctl/internal/ops"
)

func client(t *testing.T) *bugsink.ClientWithResponses {
	t.Helper()
	c := config.Context{URL: os.Getenv("BUGSINK_URL"), Token: os.Getenv("BUGSINK_TOKEN")}
	if c.URL == "" || c.Token == "" {
		t.Fatal("BUGSINK_URL and BUGSINK_TOKEN must be set")
	}
	api, err := ops.NewClient(c, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	return api
}

func TestProjectsList(t *testing.T) {
	res, err := client(t).ProjectsListWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ops.HTTPError(res.StatusCode(), res.Body); err != nil {
		t.Fatal(err)
	}
	if res.JSON200 == nil {
		t.Fatalf("no JSON body: %s", res.Body)
	}
}
