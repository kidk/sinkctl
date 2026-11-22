package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/ops"
)

func newIssuesCmd(g *globals) *cobra.Command {
	var listIn ops.ListIssuesIn
	list := listCmd(g, "list", "List issues", func(ctx context.Context, o *ops.Ops, c string) (ops.Page[bugsink.Issue], error) {
		listIn.Cursor = c
		return o.ListIssues(ctx, listIn)
	})
	f := list.Flags()
	f.IntVar(&listIn.Project, "project", 0, "only issues of this project ID")
	f.StringVarP(&listIn.Query, "query", "q", "", "title text plus key:value tag filters")
	f.StringVar(&listIn.State, "state", "", "all, open, unresolved, resolved or muted (default all)")
	f.StringVar(&listIn.Sort, "sort", "", "digest_order (needs --project), digested_event_count or last_seen")
	f.StringVar(&listIn.Order, "order", "", "asc or desc")

	get := opCmd(g, "get <id>", "Show an issue by UUID or friendly ID", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.Issue, error) {
		return o.GetIssue(ctx, ops.GetIssueIn{ID: args[0]})
	})

	var countsIn ops.IssueEventCountsIn
	counts := opCmd(g, "event-counts <id>", "Hourly event counts for an issue", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.EventCounts, error) {
		countsIn.ID = args[0]
		return o.IssueEventCounts(ctx, countsIn)
	})
	rangeFlags(counts, &countsIn.Start, &countsIn.End)

	return groupCmd("issues", "Read issues", list, get, counts)
}
