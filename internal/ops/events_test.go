package ops

import (
	"fmt"
	"net/http"
	"testing"
)

const eventUUID = "6f1c2d3e-0000-4000-8000-000000000002"

func TestListEventsRequiresIssue(t *testing.T) {
	_, err := unreachable(t).ListEvents(t.Context(), ListEventsIn{})
	wantKind(t, err, KindUsage)
}

func TestGetEventBadID(t *testing.T) {
	_, err := unreachable(t).GetEvent(t.Context(), GetEventIn{ID: "P-1"})
	wantKind(t, err, KindUsage)
}

func TestEventStacktrace(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/canonical/0/events/"+eventUUID+"/stacktrace/" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, "# ValueError\n")
	})
	s, err := o.EventStacktrace(t.Context(), GetEventIn{ID: eventUUID})
	if err != nil {
		t.Fatal(err)
	}
	if s.Text() != "# ValueError\n" {
		t.Errorf("text = %q", s.Text())
	}
}

func TestLookupEvent(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "event_id="+eventUUID+"&project=4" {
			t.Errorf("query = %q", got)
		}
		fmt.Fprintf(w, `{"id":%q,"event_id":%q,"issue":%q,"project":4,"grouping":1,"digest_order":1,
			"ingested_at":"2026-01-02T15:00:00Z","digested_at":"2026-01-02T15:00:00Z","timestamp":"2026-01-02T15:00:00Z","data":{"k":1}}`,
			eventUUID, eventUUID, eventUUID)
	})
	e, err := o.LookupEvent(t.Context(), LookupEventIn{Project: 4, EventID: eventUUID})
	if err != nil {
		t.Fatal(err)
	}
	if e.Project != 4 || e.Data == nil {
		t.Errorf("event = %+v", e)
	}
}
