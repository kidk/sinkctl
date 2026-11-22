package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kidk/sinkctl/internal/config"
	"github.com/kidk/sinkctl/internal/ops"
)

// connect serves h as Bugsink behind an in-memory MCP session.
func connect(t *testing.T, h http.HandlerFunc) *mcp.ClientSession {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	o, err := ops.New(config.Context{URL: srv.URL, Token: "tok"}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := NewServer(o).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func text(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestVersionTool(t *testing.T) {
	cs := connect(t, nil)
	res := call(t, cs, "version", nil)
	got, ok := res.StructuredContent.(map[string]any)
	if res.IsError || !ok || got["version"] != "dev" {
		t.Fatalf("result = %+v", res)
	}
}

func TestToolsRegistered(t *testing.T) {
	res, err := connect(t, nil).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != 11 {
		t.Errorf("got %d tools", len(res.Tools))
	}
}

func TestProjectsList(t *testing.T) {
	cs := connect(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "c1" {
			t.Errorf("cursor = %q", r.URL.Query().Get("cursor"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"next":"http://x/?cursor=c2","results":[{"id":1,"name":"a","slug":"a","visibility":"team_members",
			"team":"6f1c2d3e-0000-4000-8000-000000000001"}]}`)
	})
	res := call(t, cs, "projects_list", map[string]any{"cursor": "c1"})
	got, ok := res.StructuredContent.(map[string]any)
	if res.IsError || !ok || got["next_cursor"] != "c2" {
		t.Fatalf("result = %+v %s", res, text(res))
	}
}

func TestToolError(t *testing.T) {
	cs := connect(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	res := call(t, cs, "issues_get", map[string]any{"id": "P-9"})
	if !res.IsError || !strings.HasPrefix(text(res), `{"error":{"kind":"not_found","message":"issue P-9: 404 Not Found"`) {
		t.Errorf("result = %+v %s", res, text(res))
	}
}

func TestStacktraceText(t *testing.T) {
	cs := connect(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, "# boom\n")
	})
	res := call(t, cs, "events_stacktrace", map[string]any{"id": "6f1c2d3e-0000-4000-8000-000000000002"})
	if res.IsError || text(res) != "# boom\n" {
		t.Errorf("result = %+v %s", res, text(res))
	}
}

func TestToolErr(t *testing.T) {
	err := toolErr(&ops.Error{Kind: ops.KindNotFound, Message: "issue 9 not found", Status: 404})
	want := `{"error":{"kind":"not_found","message":"issue 9 not found","status":404}}`
	if err.Error() != want {
		t.Errorf("got %s, want %s", err, want)
	}
}
