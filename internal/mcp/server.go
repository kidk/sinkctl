// Package mcp exposes sinkctl ops as MCP tools.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kidk/sinkctl/internal/ops"
	"github.com/kidk/sinkctl/internal/version"
)

// NewServer registers all tools; API tools call o.
func NewServer(o *ops.Ops) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "sinkctl", Version: version.Version}, nil)
	addTool(s, "version", "Show sinkctl build information",
		func(context.Context, struct{}) (ops.VersionInfo, error) { return ops.Version(), nil })

	addTool(s, "projects_list", "List projects. Paginated: pass next_cursor back as cursor.", o.ListProjects)
	addTool(s, "projects_get", "Get a project by ID.", o.GetProject)
	addTool(s, "projects_event_counts", "Hourly event counts for a project, last 24h by default.", o.ProjectEventCounts)

	addTool(s, "issues_list", "List issues, optionally filtered by project, state or search query. Paginated: pass next_cursor back as cursor.", o.ListIssues)
	addTool(s, "issues_get", "Get an issue by UUID or friendly ID.", o.GetIssue)
	addTool(s, "issues_event_counts", "Hourly event counts for an issue, last 24h by default.", o.IssueEventCounts)

	addTool(s, "events_list", "List events of an issue, without the full payload. Paginated: pass next_cursor back as cursor.", o.ListEvents)
	addTool(s, "events_get", "Get an event by Bugsink UUID, including the full payload.", o.GetEvent)
	addTool(s, "events_stacktrace", "Get an event's stacktrace as Markdown; much smaller than events_get.", o.EventStacktrace)
	addTool(s, "events_lookup", "Find an event by project ID and SDK event_id, including the full payload.", o.LookupEvent)
	return s
}

// API types use uuid.UUID ([16]byte), which marshals as a string.
var schemaOpts = &jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[uuid.UUID](): {Type: "string", Format: "uuid"},
}}

// addTool registers a read-only tool; Text() outputs also go out as plain text content.
func addTool[In, Out any](s *mcp.Server, name, desc string, h func(context.Context, In) (Out, error)) {
	out, err := jsonschema.For[Out](schemaOpts)
	if err != nil {
		panic(err) // static types; fails every test run if broken
	}
	t := &mcp.Tool{Name: name, Description: desc, OutputSchema: out, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}
	mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		v, err := h(ctx, in)
		if err != nil {
			return nil, v, toolErr(err)
		}
		if tx, ok := any(v).(interface{ Text() string }); ok {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: tx.Text()}}}, v, nil
		}
		return nil, v, nil
	})
}

// toolErr puts the CLI's JSON error envelope in the tool error text; the SDK sets IsError.
func toolErr(err error) error {
	var e *ops.Error
	if !errors.As(err, &e) {
		return err
	}
	b, mErr := json.Marshal(map[string]*ops.Error{"error": e})
	if mErr != nil {
		return err
	}
	return errors.New(string(b))
}
