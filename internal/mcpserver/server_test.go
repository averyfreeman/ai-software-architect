package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerListsBoundedArchitectureTools(t *testing.T) {
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := NewServer()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	names := []string{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		schemaBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("tool %q schema could not be marshaled: %v", tool.Name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("tool %q has invalid input schema: %v", tool.Name, err)
		}
		names = append(names, tool.Name)
	}
	wanted := []string{"validate_architecture_contract", "scan_architecture_artifact", "validate_architecture_bundle", "list_architecture_decisions", "analyze_source_dependencies"}
	for _, name := range wanted {
		found := false
		for _, actual := range names {
			if actual == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing MCP tool %q in %v", name, names)
		}
	}
	response, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "validate_architecture_contract", Arguments: map[string]any{"content": "schema_version: 1.0.0\nrevision: 1\nscope: sample\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Content) != 1 || !strings.Contains(response.Content[0].(*mcp.TextContent).Text, `"valid": true`) {
		t.Fatalf("response = %#v", response.Content)
	}
	bundleResponse, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "validate_architecture_bundle",
		Arguments: map[string]any{
			"contract":        "schema_version: 1.0.0\nrevision: 1\nscope: sample\n",
			"decisions":       []any{},
			"project_context": "Project context",
			"coding_handoff":  "Coding handoff",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundleResponse.Content) != 1 || !strings.Contains(bundleResponse.Content[0].(*mcp.TextContent).Text, `"missing_decisions"`) {
		t.Fatalf("bundle response = %#v", bundleResponse.Content)
	}
}
