package mcp_test

import (
	"context"
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/mcp"
	"{{.ModulePath}}/internal/validation"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The add-a-tool example from the MCP sections of README.md and AGENTS.md:
// a domain tool in under 20 lines. Inside internal/mcp the same lines go into
// registerTools, without the mcp. prefixes. Keep the three copies identical.

type nameInput struct {
	Name string `json:"name" jsonschema:"the new display name"`
}

func registerNameTool(server *mcpsdk.Server, services mcp.Services) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "set_display_name",
		Description: "Changes the display name of the user this token acts for. Needs a token with read and write access.",
	}, mcp.Write(func(ctx context.Context, caller mcp.Caller, input nameInput) (api.UserResponse, error) {
		err := validation.ValidateName(input.Name)
		if err != nil {
			return api.UserResponse{}, mcp.InvalidField("name", api.FieldMessage(err))
		}
		user, err := services.Users.UpdateName(caller.User.ID, input.Name)
		if err != nil {
			return api.UserResponse{}, err
		}
		return api.UserResponse{User: api.NewUser(user)}, nil
	}))
}

func TestAddedToolRunsWithTheTokenScopes(t *testing.T) {
	env := newEnvironment(t)
	owner := seedOwner(t, env, "namer")
	server := mcp.NewServer(env.services)
	registerNameTool(server, env.services)
	readToken, _ := createToken(t, env, owner)
	writeToken, _ := createToken(t, env, owner, "read", "write")

	reader := connectMemory(t, server, owner, readToken)
	toolError(t, call(t, reader, "set_display_name", map[string]any{"name": "Grace Hopper"}), "insufficient_scope")

	writer := connectMemory(t, server, owner, writeToken)
	tools, err := writer.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTool(tools.Tools, "set_display_name") || !hasTool(tools.Tools, "whoami") {
		t.Fatalf("tools = %v", tools.Tools)
	}
	updated := output[api.UserResponse](t, call(t, writer, "set_display_name", map[string]any{"name": "Grace Hopper"}))
	if updated.User.Name != "Grace Hopper" {
		t.Fatalf("user = %#v", updated.User)
	}
	stored, err := env.services.Users.ByID(owner.user.ID)
	if err != nil || stored.Name != "Grace Hopper" {
		t.Fatalf("stored name = %v, %v", stored, err)
	}
	fields := toolError(t, call(t, writer, "set_display_name", map[string]any{"name": ""}), "validation_failed").Fields
	if fields["name"] == "" {
		t.Fatalf("fields = %v", fields)
	}
	result, err := writer.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: "set_display_name", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatalf("missing name: result %v, error %v", result, err)
	}
}

func hasTool(tools []*mcpsdk.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
