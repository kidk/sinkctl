package ops

import (
	"fmt"
	"net/http"
	"testing"
)

func TestListIssuesParams(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "project=2&q=boom&state=open" {
			t.Errorf("query = %q", got)
		}
		fmt.Fprint(w, `{"results":[{"id":"6f1c2d3e-0000-4000-8000-000000000001","friendly_id":"P-1","project":2,
			"calculated_type":"ValueError","calculated_value":"boom","digest_order":1,"digested_event_count":3,
			"first_seen":"2026-01-02T15:00:00Z","last_seen":"2026-01-02T15:00:00Z","is_resolved":false,"is_muted":true}]}`)
	})
	p, err := o.ListIssues(t.Context(), ListIssuesIn{Project: 2, Query: "boom", State: "open"})
	if err != nil {
		t.Fatal(err)
	}
	_, rows := p.Table()
	if want := []string{"P-1", "ValueError: boom", "3", "2026-01-02T15:00:00Z", "muted"}; fmt.Sprint(rows) != fmt.Sprint([][]string{want}) {
		t.Errorf("rows = %v", rows)
	}
}

func TestListIssuesBadEnum(t *testing.T) {
	_, err := unreachable(t).ListIssues(t.Context(), ListIssuesIn{State: "opne"})
	wantKind(t, err, KindUsage)
}

func TestGetIssueRequiresID(t *testing.T) {
	_, err := unreachable(t).GetIssue(t.Context(), GetIssueIn{})
	wantKind(t, err, KindUsage)
}

func TestIssueEventCountsAuth(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := o.IssueEventCounts(t.Context(), IssueEventCountsIn{ID: "P-1"})
	wantKind(t, err, KindAuth)
}
