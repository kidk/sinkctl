// Package mcp exposes sinkctl ops as MCP tools.
package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kidk/sinkctl/internal/ops"
	"github.com/kidk/sinkctl/internal/version"
)

// NewServer registers tools; o backs the API tools (none yet).
func NewServer(o *ops.Ops) *mcp.Server {
	_ = o
	s := mcp.NewServer(&mcp.Implementation{Name: "sinkctl", Version: version.Version}, nil)
	addTool(s, &mcp.Tool{Name: "version", Description: "Show sinkctl build information"},
		func(context.Context, struct{}) (ops.VersionInfo, error) { return ops.Version(), nil })
	return s
}

func addTool[In, Out any](s *mcp.Server, t *mcp.Tool, h func(context.Context, In) (Out, error)) {
	mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		out, err := h(ctx, in)
		return nil, out, toolErr(err)
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
