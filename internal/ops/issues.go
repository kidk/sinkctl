package ops

import (
	"context"
	"strconv"
	"time"

	"github.com/kidk/sinkctl/internal/bugsink"
)

type ListIssuesIn struct {
	Project int    `json:"project,omitempty" jsonschema:"only issues of this project ID; default all accessible projects"`
	Query   string `json:"q,omitempty" jsonschema:"plain-text title search plus key:value tag filters; quote values with spaces"`
	State   string `json:"state,omitempty" jsonschema:"all (default), open, unresolved, resolved or muted"`
	Sort    string `json:"sort,omitempty" jsonschema:"digest_order (single project only), digested_event_count or last_seen"`
	Order   string `json:"order,omitempty" jsonschema:"asc or desc"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"next_cursor from a previous page"`
}

var issueHeader = []string{"ID", "TITLE", "EVENTS", "LAST SEEN", "STATE"}

func issueRow(i bugsink.Issue) []string {
	title := fmtPtr(i.CalculatedType)
	if v := fmtPtr(i.CalculatedValue); v != "" {
		title += ": " + v
	}
	return []string{
		fmtPtr(i.FriendlyId), title, strconv.FormatInt(i.DigestedEventCount, 10),
		i.LastSeen.Format(time.RFC3339), issueState(i),
	}
}

func issueState(i bugsink.Issue) string {
	switch {
	case i.IsResolved != nil && *i.IsResolved:
		return "resolved"
	case i.IsMuted != nil && *i.IsMuted:
		return "muted"
	}
	return "open"
}

func (o *Ops) ListIssues(ctx context.Context, in ListIssuesIn) (Page[bugsink.Issue], error) {
	res, err := o.api.IssuesListWithResponse(ctx, &bugsink.IssuesListParams{
		Project: ptr(in.Project),
		Q:       ptr(in.Query),
		State:   ptr(bugsink.IssuesListParamsState(in.State)),
		Sort:    ptr(bugsink.IssuesListParamsSort(in.Sort)),
		Order:   ptr(bugsink.IssuesListParamsOrder(in.Order)),
		Cursor:  ptr(in.Cursor),
	})
	if err != nil {
		return Page[bugsink.Issue]{}, transportErr("issues", err)
	}
	v, err := decoded("issues", res.StatusCode(), res.Body, res.JSON200)
	return newPage(v.Results, v.Next, issueHeader, issueRow), err
}

type GetIssueIn struct {
	ID string `json:"id" jsonschema:"issue UUID or friendly ID like PROJECT-123"`
}

type Issue struct{ bugsink.Issue }

func (i Issue) Table() ([]string, [][]string) {
	return append([]string{"UUID"}, issueHeader...), [][]string{append([]string{fmtPtr(i.Id)}, issueRow(i.Issue)...)}
}

func (o *Ops) GetIssue(ctx context.Context, in GetIssueIn) (Issue, error) {
	if in.ID == "" {
		return Issue{}, usagef("issue id is required")
	}
	what := "issue " + in.ID
	res, err := o.api.IssuesRetrieveWithResponse(ctx, in.ID)
	if err != nil {
		return Issue{}, transportErr(what, err)
	}
	v, err := decoded(what, res.StatusCode(), res.Body, res.JSON200)
	return Issue{v}, err
}

type IssueEventCountsIn struct {
	ID    string `json:"id" jsonschema:"issue UUID or friendly ID"`
	Start string `json:"start,omitempty" jsonschema:"inclusive UTC hour, RFC 3339; default 24h before end"`
	End   string `json:"end,omitempty" jsonschema:"exclusive UTC hour, RFC 3339; default end of current hour"`
}

func (o *Ops) IssueEventCounts(ctx context.Context, in IssueEventCountsIn) (EventCounts, error) {
	if in.ID == "" {
		return EventCounts{}, usagef("issue id is required")
	}
	what := "issue " + in.ID + " event counts"
	start, end, err := parseRange(in.Start, in.End)
	if err != nil {
		return EventCounts{}, err
	}
	res, err := o.api.IssuesEventCountsRetrieveWithResponse(ctx, in.ID,
		&bugsink.IssuesEventCountsRetrieveParams{Start: start, End: end})
	if err != nil {
		return EventCounts{}, transportErr(what, err)
	}
	v, err := decoded(what, res.StatusCode(), res.Body, res.JSON200)
	return EventCounts{v}, err
}
