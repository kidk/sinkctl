package ops

import (
	"context"
	"strconv"

	"github.com/kidk/sinkctl/internal/bugsink"
)

type ListProjectsIn struct {
	Team   string `json:"team,omitempty" jsonschema:"only projects of this team UUID"`
	Cursor string `json:"cursor,omitempty" jsonschema:"next_cursor from a previous page"`
}

var projectHeader = []string{"ID", "SLUG", "NAME", "VISIBILITY", "EVENTS"}

func projectRow(p bugsink.ProjectList) []string {
	return []string{fmtPtr(p.Id), p.Slug, p.Name, string(p.Visibility), fmtPtr(p.DigestedEventCount)}
}

func (o *Ops) ListProjects(ctx context.Context, in ListProjectsIn) (Page[bugsink.ProjectList], error) {
	params := &bugsink.ProjectsListParams{Cursor: ptr(in.Cursor)}
	if in.Team != "" {
		id, err := parseUUID("team", in.Team)
		if err != nil {
			return Page[bugsink.ProjectList]{}, err
		}
		params.Team = &id
	}
	res, err := o.api.ProjectsListWithResponse(ctx, params)
	if err != nil {
		return Page[bugsink.ProjectList]{}, transportErr("projects", err)
	}
	v, err := decoded("projects", res.HTTPResponse, res.Body, res.JSON200)
	if err != nil {
		return Page[bugsink.ProjectList]{}, err
	}
	return newPage(v.Results, v.Next, projectHeader, projectRow)
}

type GetProjectIn struct {
	ID int `json:"id" jsonschema:"project ID"`
}

type Project struct{ bugsink.ProjectDetail }

func (p Project) Table() ([]string, [][]string) {
	return []string{"ID", "SLUG", "NAME", "VISIBILITY", "EVENTS", "DSN"}, [][]string{{
		fmtPtr(p.Id), p.Slug, p.Name, string(p.Visibility), fmtPtr(p.DigestedEventCount), fmtPtr(p.Dsn),
	}}
}

func (o *Ops) GetProject(ctx context.Context, in GetProjectIn) (Project, error) {
	what := "project " + strconv.Itoa(in.ID)
	res, err := o.api.ProjectsRetrieveWithResponse(ctx, in.ID, nil)
	if err != nil {
		return Project{}, transportErr(what, err)
	}
	v, err := decoded(what, res.HTTPResponse, res.Body, res.JSON200)
	return Project{v}, err
}

type ProjectEventCountsIn struct {
	ID    int    `json:"id" jsonschema:"project ID"`
	Start string `json:"start,omitempty" jsonschema:"inclusive UTC hour, RFC 3339; default 24h before end"`
	End   string `json:"end,omitempty" jsonschema:"exclusive UTC hour, RFC 3339; default end of current hour"`
}

func (o *Ops) ProjectEventCounts(ctx context.Context, in ProjectEventCountsIn) (EventCounts, error) {
	what := "project " + strconv.Itoa(in.ID) + " event counts"
	start, end, err := parseRange(in.Start, in.End)
	if err != nil {
		return EventCounts{}, err
	}
	res, err := o.api.ProjectsEventCountsRetrieveWithResponse(ctx, in.ID,
		&bugsink.ProjectsEventCountsRetrieveParams{Start: start, End: end})
	if err != nil {
		return EventCounts{}, transportErr(what, err)
	}
	v, err := decoded(what, res.HTTPResponse, res.Body, res.JSON200)
	return EventCounts{v}, err
}
