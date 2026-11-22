package ops

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kidk/sinkctl/internal/config"
)

// fake serves h as Bugsink and returns Ops pointed at it.
func fake(t *testing.T, h http.HandlerFunc) *Ops {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // generated client only decodes JSON
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	o, err := New(config.Context{URL: srv.URL, Token: "tok"}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return o
}

// unreachable fails the test if any request is made.
func unreachable(t *testing.T) *Ops {
	return fake(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected request") })
}

func wantKind(t *testing.T, err error, k Kind) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Kind != k {
		t.Fatalf("got %v, want kind %q", err, k)
	}
}

func TestAuthHeaders(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "sinkctl/dev" {
			t.Errorf("User-Agent = %q", got)
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := o.ListProjects(t.Context(), ListProjectsIn{}); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkError(t *testing.T) {
	o, err := New(config.Context{URL: "http://127.0.0.1:1", Token: "t"}, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.ListProjects(t.Context(), ListProjectsIn{})
	wantKind(t, err, KindNetwork)
}
