package ops

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListProjectsTeamFilter(t *testing.T) {
	const team = "6f1c2d3e-0000-4000-8000-000000000001"
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("team"); got != team {
			t.Errorf("team = %q", got)
		}
		fmt.Fprint(w, `{"results":[]}`)
	})
	p, err := o.ListProjects(t.Context(), ListProjectsIn{Team: team})
	if err != nil {
		t.Fatal(err)
	}
	if p.Results == nil {
		t.Error("results must be [] not null")
	}
}

func TestListProjectsBadTeam(t *testing.T) {
	_, err := unreachable(t).ListProjects(t.Context(), ListProjectsIn{Team: "nope"})
	wantKind(t, err, KindUsage)
}

func TestGetProjectNotFound(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail":"No Project matches the given query."}`)
	})
	_, err := o.GetProject(t.Context(), GetProjectIn{ID: 9})
	var e *Error
	if !errors.As(err, &e) || e.Kind != KindNotFound || e.Status != 404 || !strings.HasPrefix(e.Message, "project 9: 404 Not Found") {
		t.Errorf("error = %+v", err)
	}
}

func TestUnexpectedBodySnippet(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<!doctype html>login")
	})
	_, err := o.GetProject(t.Context(), GetProjectIn{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "(text/html): <!doctype html>login") {
		t.Errorf("err = %v", err)
	}
}

func TestProjectEventCounts(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/canonical/0/projects/3/event-counts/" || r.URL.Query().Get("start") != "2026-01-02T15:00:00Z" {
			t.Errorf("url = %s", r.URL)
		}
		fmt.Fprint(w, `{"start":"2026-01-02T15:00:00Z","end":"2026-01-02T16:00:00Z","interval_seconds":3600,
			"buckets":[{"timestamp":"2026-01-02T15:00:00Z","count":4}]}`)
	})
	c, err := o.ProjectEventCounts(t.Context(), ProjectEventCountsIn{ID: 3, Start: "2026-01-02T15:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if _, rows := c.Table(); len(rows) != 1 || rows[0][1] != "4" {
		t.Errorf("rows = %v", rows)
	}
}

func TestEventCountsBadTime(t *testing.T) {
	_, err := unreachable(t).ProjectEventCounts(t.Context(), ProjectEventCountsIn{ID: 3, End: "yesterday"})
	wantKind(t, err, KindUsage)
}
