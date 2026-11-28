//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kidk/sinkctl/internal/cli"
	sinkmcp "github.com/kidk/sinkctl/internal/mcp"
)

func sinkctl(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // creds only from env
	var out, errb bytes.Buffer
	code = cli.Run(t.Context(), args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestCLIReads(t *testing.T) {
	project := strconv.Itoa(seed.project)
	cases := [][]string{
		{"projects", "list", "--all"},
		{"projects", "get", project},
		{"projects", "event-counts", project},
		{"issues", "list", "--project", project, "--state", "open"},
		{"issues", "get", seed.friendlyID},
		{"issues", "event-counts", seed.issue},
		{"events", "list", "--issue", seed.issue},
		{"events", "get", seed.event},
		{"events", "stacktrace", seed.event},
		{"events", "lookup", "--project", project, "--event-id", seed.sdkEventID},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			code, out, stderr := sinkctl(t, args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if !json.Valid([]byte(out)) {
				t.Errorf("stdout not JSON: %s", out)
			}
		})
	}
}

// config login's URL probe assumes a token-less request gets a JSON 401/403 here.
func TestUnauthenticatedProjectsIsJSONDenial(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, os.Getenv("BUGSINK_URL")+"/api/canonical/0/projects/", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close() //nolint:errcheck // test
	body, _ := io.ReadAll(res.Body)
	if (res.StatusCode != http.StatusUnauthorized && res.StatusCode != http.StatusForbidden) || !json.Valid(body) {
		t.Errorf("got %d %s", res.StatusCode, body)
	}
}

func TestCLIIssuesListFindsSeed(t *testing.T) {
	_, out, _ := sinkctl(t, "issues", "list", "--project", strconv.Itoa(seed.project), "-q", "e2e boom")
	if !strings.Contains(out, seed.issue) {
		t.Errorf("seeded issue missing: %s", out)
	}
}

func TestCLIStacktraceText(t *testing.T) {
	_, out, _ := sinkctl(t, "events", "stacktrace", seed.event, "-o", "table")
	if !strings.Contains(out, "ValueError") {
		t.Errorf("stacktrace: %s", out)
	}
}

func TestCLINotFound(t *testing.T) {
	code, _, stderr := sinkctl(t, "issues", "get", "00000000-0000-4000-8000-000000000000")
	if code != 4 || !strings.Contains(stderr, `"kind":"not_found"`) {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

func TestMCPReads(t *testing.T) {
	ctx := t.Context()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := sinkmcp.NewServer(seed.ops).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	cases := map[string]map[string]any{
		"projects_list":         {},
		"projects_get":          {"id": seed.project},
		"projects_event_counts": {"id": seed.project},
		"issues_list":           {"project": seed.project},
		"issues_get":            {"id": seed.friendlyID},
		"issues_event_counts":   {"id": seed.issue},
		"events_list":           {"issue": seed.issue},
		"events_get":            {"id": seed.event},
		"events_stacktrace":     {"id": seed.event},
		"events_lookup":         {"project": seed.project, "event_id": seed.sdkEventID},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
			if err != nil {
				t.Fatal(err)
			}
			if res.IsError || res.StructuredContent == nil {
				t.Errorf("result: %+v", res.Content)
			}
		})
	}
}
