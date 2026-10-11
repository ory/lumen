package cmd

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Claude Code defers every MCP tool behind ToolSearch by default, so the
// model never sees semantic_search's description unless the server marks it
// to load upfront (anthropic/alwaysLoad, Claude Code >= 2.1.285). Benchmarks
// on 2026-10-11 showed Sonnet 5.5 calling the deferred tool in 1 of 12 tasks.
func TestSemanticSearchToolLoadsUpfront(t *testing.T) {
	server := newMCPServer(&indexerCache{})
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]any{}
	for _, tool := range tools.Tools {
		got[tool.Name] = tool.Meta["anthropic/alwaysLoad"]
	}
	if got["semantic_search"] != true {
		t.Errorf("semantic_search must advertise anthropic/alwaysLoad=true so Claude Code loads it upfront; got %v", got["semantic_search"])
	}
	for _, name := range []string{"index_status", "health_check"} {
		if _, ok := got[name]; !ok {
			t.Errorf("tool %s missing from tools/list", name)
		} else if got[name] == true {
			t.Errorf("%s should stay deferred to keep context small", name)
		}
	}
}
