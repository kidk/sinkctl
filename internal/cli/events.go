package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/bugsink"
	"github.com/kidk/sinkctl/internal/ops"
)

func newEventsCmd(g *globals) *cobra.Command {
	var listIn ops.ListEventsIn
	list := listCmd(g, "list", "List events of an issue", func(ctx context.Context, o *ops.Ops, c string) (ops.Page[bugsink.EventList], error) {
		listIn.Cursor = c
		return o.ListEvents(ctx, listIn)
	})
	f := list.Flags()
	f.StringVar(&listIn.Issue, "issue", "", "issue UUID (required)")
	f.StringVarP(&listIn.Query, "query", "q", "", "title text plus key:value tag filters")
	f.StringVar(&listIn.Order, "order", "", "asc or desc (default desc)")

	get := opCmd(g, "get <id>", "Show an event with its full payload", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.Event, error) {
		return o.GetEvent(ctx, ops.GetEventIn{ID: args[0]})
	})

	stack := opCmd(g, "stacktrace <id>", "Print an event's stacktrace as Markdown", 1, func(ctx context.Context, o *ops.Ops, args []string) (ops.Stacktrace, error) {
		return o.EventStacktrace(ctx, ops.GetEventIn{ID: args[0]})
	})

	var lookupIn ops.LookupEventIn
	lookup := opCmd(g, "lookup", "Find an event by project and SDK event_id", 0, func(ctx context.Context, o *ops.Ops, _ []string) (ops.Event, error) {
		return o.LookupEvent(ctx, lookupIn)
	})
	lookup.Flags().IntVar(&lookupIn.Project, "project", 0, "project ID (required)")
	lookup.Flags().StringVar(&lookupIn.EventID, "event-id", "", "SDK event_id UUID (required)")

	return groupCmd("events", "Read events", list, get, stack, lookup)
}
