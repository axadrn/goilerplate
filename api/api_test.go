package api

import (
	"encoding/json"
	"testing"
)

func TestGenerationAnswersJSON(t *testing.T) {
	encoded, err := json.Marshal(GenerationAnswers{Workspaces: true, Framework: "headless", API: true, MCP: true})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["workspaces"]) != "true" {
		t.Fatalf("workspaces = %s", fields["workspaces"])
	}
	if string(fields["api"]) != "true" {
		t.Fatalf("api = %s", fields["api"])
	}
	if string(fields["mcp"]) != "true" {
		t.Fatalf("mcp = %s", fields["mcp"])
	}
	if string(fields["framework"]) != `"headless"` {
		t.Fatalf("framework = %s", fields["framework"])
	}
	if len(fields) != 13 {
		t.Fatalf("generation answer fields = %d, want 13", len(fields))
	}
}
