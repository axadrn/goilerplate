package api

import (
	"encoding/json"
	"testing"
)

func TestGenerationAnswersJSON(t *testing.T) {
	encoded, err := json.Marshal(GenerationAnswers{Workspaces: true, Demo: true})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"workspaces", "demo"} {
		if string(fields[name]) != "true" {
			t.Fatalf("%s = %s", name, fields[name])
		}
	}
}
