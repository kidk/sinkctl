package ops

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/kidk/sinkctl/internal/bugsink"
)

func TestAllFollowsCursors(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, `{"next":"http://x/api/canonical/0/projects/?cursor=c2","results":[{"id":1,"name":"a","slug":"a","visibility":"team_members"}]}`)
		case "c2":
			fmt.Fprint(w, `{"results":[{"id":2,"name":"b","slug":"b","visibility":"team_members"}]}`)
		}
	})
	one, err := o.ListProjects(t.Context(), ListProjectsIn{})
	if err != nil {
		t.Fatal(err)
	}
	if one.NextCursor != "c2" || len(one.Results) != 1 {
		t.Fatalf("first page = %+v", one)
	}

	all, err := All(t.Context(), "", func(ctx context.Context, c string) (Page[bugsink.ProjectList], error) {
		return o.ListProjects(ctx, ListProjectsIn{Cursor: c})
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Results) != 2 || all.NextCursor != "" {
		t.Fatalf("all = %+v", all)
	}
	if _, rows := all.Table(); len(rows) != 2 || rows[1][1] != "b" {
		t.Errorf("rows = %v", rows)
	}
}

func TestAllStopsOnError(t *testing.T) {
	o := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "c2" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"next":"http://x/?cursor=c2","results":[]}`)
	})
	_, err := All(t.Context(), "", func(ctx context.Context, c string) (Page[bugsink.ProjectList], error) {
		return o.ListProjects(ctx, ListProjectsIn{Cursor: c})
	})
	wantKind(t, err, KindNotFound)
}
