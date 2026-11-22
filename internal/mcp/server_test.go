package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kidk/sinkctl/internal/ops"
)

func TestVersionTool(t *testing.T) {
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := NewServer(nil).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "version"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	got, ok := res.StructuredContent.(map[string]any)
	if !ok || got["version"] != "dev" {
		t.Fatalf("structured content = %#v", res.StructuredContent)
	}
}

func TestToolErr(t *testing.T) {
	err := toolErr(&ops.Error{Kind: ops.KindNotFound, Message: "issue 9 not found", Status: 404})
	want := `{"error":{"kind":"not_found","message":"issue 9 not found","status":404}}`
	if err.Error() != want {
		t.Errorf("got %s, want %s", err, want)
	}
}
