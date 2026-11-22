package ops

import (
	"context"
	"strconv"
	"time"

	"github.com/kidk/sinkctl/internal/bugsink"
)

type ListEventsIn struct {
	Issue  string `json:"issue" jsonschema:"issue UUID"`
	Query  string `json:"q,omitempty" jsonschema:"plain-text title search plus key:value tag filters; quote values with spaces"`
	Order  string `json:"order,omitempty" jsonschema:"asc or desc (default) by digest order"`
	Cursor string `json:"cursor,omitempty" jsonschema:"next_cursor from a previous page"`
}

var eventHeader = []string{"ID", "EVENT ID", "TIMESTAMP", "DIGEST ORDER"}

func eventRow(e bugsink.EventList) []string {
	return []string{fmtPtr(e.Id), fmtPtr(e.EventId), e.Timestamp.Format(time.RFC3339), strconv.FormatInt(e.DigestOrder, 10)}
}

func (o *Ops) ListEvents(ctx context.Context, in ListEventsIn) (Page[bugsink.EventList], error) {
	if in.Issue == "" {
		return Page[bugsink.EventList]{}, usagef("issue is required")
	}
	res, err := o.api.EventsListWithResponse(ctx, &bugsink.EventsListParams{
		Issue:  in.Issue,
		Q:      ptr(in.Query),
		Order:  ptr(bugsink.EventsListParamsOrder(in.Order)),
		Cursor: ptr(in.Cursor),
	})
	if err != nil {
		return Page[bugsink.EventList]{}, transportErr("events", err)
	}
	v, err := decoded("events", res.StatusCode(), res.Body, res.JSON200)
	return newPage(v.Results, v.Next, eventHeader, eventRow), err
}

type GetEventIn struct {
	ID string `json:"id" jsonschema:"Bugsink event UUID (the id field, not the SDK event_id)"`
}

type Event struct{ bugsink.EventDetail }

func (e Event) Table() ([]string, [][]string) {
	return []string{"ID", "EVENT ID", "TIMESTAMP", "DIGEST ORDER", "ISSUE"}, [][]string{{
		fmtPtr(e.Id), fmtPtr(e.EventId), e.Timestamp.Format(time.RFC3339),
		strconv.FormatInt(e.DigestOrder, 10), e.Issue.String(),
	}}
}

func (o *Ops) GetEvent(ctx context.Context, in GetEventIn) (Event, error) {
	id, err := parseUUID("event id", in.ID)
	if err != nil {
		return Event{}, err
	}
	what := "event " + in.ID
	res, err := o.api.EventsRetrieveWithResponse(ctx, id)
	if err != nil {
		return Event{}, transportErr(what, err)
	}
	v, err := decoded(what, res.StatusCode(), res.Body, res.JSON200)
	return Event{v}, err
}

type Stacktrace struct {
	Stacktrace string `json:"stacktrace"`
}

func (s Stacktrace) Text() string { return s.Stacktrace }

func (o *Ops) EventStacktrace(ctx context.Context, in GetEventIn) (Stacktrace, error) {
	id, err := parseUUID("event id", in.ID)
	if err != nil {
		return Stacktrace{}, err
	}
	what := "event " + in.ID + " stacktrace"
	res, err := o.api.EventsStacktraceRetrieveWithResponse(ctx, id)
	if err != nil {
		return Stacktrace{}, transportErr(what, err)
	}
	body := string(res.Body)
	body, err = decoded(what, res.StatusCode(), res.Body, &body)
	return Stacktrace{body}, err
}

type LookupEventIn struct {
	Project int    `json:"project" jsonschema:"project ID"`
	EventID string `json:"event_id" jsonschema:"SDK-generated event_id (UUID)"`
}

func (o *Ops) LookupEvent(ctx context.Context, in LookupEventIn) (Event, error) {
	if in.Project == 0 {
		return Event{}, usagef("project is required")
	}
	id, err := parseUUID("event_id", in.EventID)
	if err != nil {
		return Event{}, err
	}
	what := "event " + in.EventID + " in project " + strconv.Itoa(in.Project)
	res, err := o.api.EventsLookupRetrieveWithResponse(ctx, &bugsink.EventsLookupRetrieveParams{EventId: id, Project: in.Project})
	if err != nil {
		return Event{}, transportErr(what, err)
	}
	v, err := decoded(what, res.StatusCode(), res.Body, res.JSON200)
	return Event{v}, err
}
