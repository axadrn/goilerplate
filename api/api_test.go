package api

import (
	"encoding/json"
	"testing"
)

func TestGenerationAnswersJSON(t *testing.T) {
	encoded, err := json.Marshal(GenerationAnswers{Workspaces: true})
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
	if len(fields) != 11 {
		t.Fatalf("generation answer fields = %d, want 11", len(fields))
	}
}
