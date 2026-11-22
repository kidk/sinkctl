package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeAPI(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestProjectsListAll(t *testing.T) {
	url := fakeAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"next":"http://x/?cursor=c2","results":[{"id":1,"name":"a","slug":"a","visibility":"team_members"}]}`)
			return
		}
		fmt.Fprint(w, `{"results":[{"id":2,"name":"b","slug":"b","visibility":"team_members"}]}`)
	})

	_, out, _ := execAPI(t, url, "projects", "list")
	var page struct {
		Results    []map[string]any `json:"results"`
		NextCursor string           `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(out), &page); err != nil || page.NextCursor != "c2" || len(page.Results) != 1 {
		t.Fatalf("one page: %v %s", err, out)
	}

	code, out, _ := execAPI(t, url, "projects", "list", "--all", "-o", "table")
	if code != 0 || !strings.Contains(out, "SLUG") || strings.Count(out, "team_members") != 2 {
		t.Errorf("exit %d, table:\n%s", code, out)
	}
}

func TestIssuesGetNotFound(t *testing.T) {
	url := fakeAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail":"Not found."}`)
	})
	code, _, stderr := execAPI(t, url, "issues", "get", "P-9")
	if code != 4 || !strings.Contains(stderr, `"kind":"not_found"`) || !strings.Contains(stderr, "issue P-9") {
		t.Errorf("exit %d, stderr %s", code, stderr)
	}
}

func TestStacktraceRawInTable(t *testing.T) {
	url := fakeAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, "# boom\n")
	})
	_, out, _ := execAPI(t, url, "events", "stacktrace", "6f1c2d3e-0000-4000-8000-000000000002", "-o", "table")
	if out != "# boom\n" {
		t.Errorf("out = %q", out)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"projects", "get", "abc"},
		{"projects", "nope"},
		{"events", "list"},
	} {
		if code, _, stderr := execAPI(t, "http://127.0.0.1:1", args...); code != 2 {
			t.Errorf("%v: exit %d, stderr %s", args, code, stderr)
		}
	}
}
