package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/ops"
)

func newProjectsCmd(g *globals) *cobra.Command {
	var listIn ops.ListProjectsIn
	list := listCmd(g, "list", "List projects", func(ctx context.Context, o *ops.Ops, c string) (ops.Page[bugsink.ProjectList], error) {
		listIn.Cursor = c
		return o.ListProjects(ctx, listIn)
	})
	list.Flags().StringVar(&listIn.Team, "team", "", "only projects of this team UUID")

	get := opCmd(g, "get <id>", "Show a project", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.Project, error) {
		id, err := intArg("project id", args[0])
		if err != nil {
			return ops.Project{}, err
		}
		return o.GetProject(ctx, ops.GetProjectIn{ID: id})
	})

	var countsIn ops.ProjectEventCountsIn
	counts := opCmd(g, "event-counts <id>", "Hourly event counts for a project", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.EventCounts, error) {
		id, err := intArg("project id", args[0])
		if err != nil {
			return ops.EventCounts{}, err
		}
		countsIn.ID = id
		return o.ProjectEventCounts(ctx, countsIn)
	})
	rangeFlags(counts, &countsIn.Start, &countsIn.End)

	return groupCmd("projects", "Read projects", list, get, counts)
}
