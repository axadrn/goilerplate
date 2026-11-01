package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Caller is who called a tool: the user and the personal API token the
// request authenticated with.
type Caller struct {
	User  *model.User
	Token *model.APIToken
}

// ToolFunc is the body of a tool. It gets the caller and the decoded input
// and returns the structured output. Errors from services are reported with
// the JSON API codes.
type ToolFunc[In, Out any] func(ctx context.Context, caller Caller, input In) (Out, error)

// Read wraps a tool that only reads data. It needs the read scope, which every
// token has.
func Read[In, Out any](tool ToolFunc[In, Out]) mcpsdk.ToolHandlerFor[In, Out] {
	return withScope(model.APITokenScopeRead, tool)
}

// Write wraps a tool that changes data. A read-only token gets the error code
// insufficient_scope.
func Write[In, Out any](tool ToolFunc[In, Out]) mcpsdk.ToolHandlerFor[In, Out] {
	return withScope(model.APITokenScopeWrite, tool)
}

func withScope[In, Out any](scope string, tool ToolFunc[In, Out]) mcpsdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input In) (*mcpsdk.CallToolResult, Out, error) {
		var empty Out
		caller := Caller{User: ctxkeys.User(ctx), Token: ctxkeys.APIToken(ctx)}
		if caller.User == nil || caller.Token == nil {
			return nil, empty, &ToolError{Code: "unauthenticated", Message: "Send a personal API token as Authorization: Bearer pat_..."}
		}
		if !caller.Token.HasScope(scope) {
			return nil, empty, &ToolError{Code: "insufficient_scope", Message: "This API token needs the " + scope + " scope. Use a token with read and write access."}
		}
		output, err := tool(ctx, caller, input)
		if err != nil {
			return nil, empty, toolError(err)
		}
		return result(output), output, nil
	}
}

// Summarizer is implemented by tool outputs that describe themselves in one
// sentence. The sentence comes before the JSON in the text content, so
// clients that only read text still get both.
type Summarizer interface {
	Summary() string
}

// result adds the summary of output to the text content. Without a summary
// the SDK fills the text content with the JSON of output.
func result(output any) *mcpsdk.CallToolResult {
	summarizer, ok := output.(Summarizer)
	if !ok {
		return nil
	}
	content := []mcpsdk.Content{&mcpsdk.TextContent{Text: summarizer.Summary()}}
	data, err := json.Marshal(output)
	if err == nil {
		content = append(content, &mcpsdk.TextContent{Text: string(data)})
	}
	return &mcpsdk.CallToolResult{Content: content}
}

// ToolError is a failed tool call. Its text is the error envelope of the JSON
// API, so agents see the same codes and field messages.
type ToolError struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e *ToolError) Error() string {
	data, err := json.Marshal(api.ErrorBody{Error: api.ErrorDetail{Code: e.Code, Message: e.Message, Fields: e.Fields}})
	if err != nil {
		return e.Message
	}
	return string(data)
}

// InvalidField returns a validation_failed error for one input field.
func InvalidField(field, message string) error {
	return &ToolError{Code: "validation_failed", Message: api.ValidationMessage, Fields: map[string]string{field: message}}
}

// toolError maps a service or repository error to the code the JSON API
// answers for it. Unknown errors are logged and reported as internal.
func toolError(err error) error {
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return toolErr
	}
	_, detail, known := api.ErrorFor(err)
	if !known {
		slog.Error("mcp tool failed", "error", err)
	}
	return &ToolError{Code: detail.Code, Message: detail.Message, Fields: detail.Fields}
}
