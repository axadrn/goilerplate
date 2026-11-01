package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
)

// APITokensHandler creates and revokes personal API tokens from the account
// settings.
type APITokensHandler struct {
	tokens *service.APITokenService
}

func NewAPITokensHandler(tokens *service.APITokenService) *APITokensHandler {
	return &APITokensHandler{tokens: tokens}
}

// Create stores a token and patches the fresh dialog, the list and the
// one-time reveal. Invalid input patches the form inside the open dialog.
func (h *APITokensHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	sse := datastar.NewSSE(w, r)
	input, form := parseAPITokenForm(r)
	if len(form.Errors) == 0 {
		token, raw, err := h.tokens.Create(user.ID, input)
		if err == nil {
			slog.Info("api token created", "token_id", token.ID, "user_id", user.ID)
			view, err := apiTokensView(r, h.tokens, newAPITokenForm(r))
			if err != nil {
				slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
				patchAPITokenToast(sse, toast.Props{Title: "Token created", Description: "Reload the page to see it in the list.", Type: toast.TypeInfo})
				patchAPITokens(sse, pages.APITokenReveal(raw), datastar.WithModeReplace())
				return
			}
			// Replacing the dialog closes it and resets the form.
			patchAPITokens(sse, pages.APITokenCreateDialog(view), datastar.WithModeReplace())
			patchAPITokens(sse, pages.APITokenList(view))
			patchAPITokens(sse, pages.APITokenReveal(raw), datastar.WithModeReplace())
			return
		}
		field, message, ok := apiTokenFormError(err)
		if !ok {
			slog.Error("failed to create api token", "error", err, "user_id", user.ID)
			patchAPITokenToast(sse, toast.Props{Title: "Token could not be created", Description: "Please try again.", Type: toast.TypeError})
			return
		}
		form.Errors[field] = message
	}
	view, err := apiTokensView(r, h.tokens, form)
	if err != nil {
		slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
		patchAPITokenToast(sse, toast.Props{Title: "Token could not be created", Description: "Please try again.", Type: toast.TypeError})
		return
	}
	patchAPITokens(sse, pages.APITokenCreateForm(view))
}

// Revoke deletes a token and patches the updated list.
func (h *APITokensHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	sse := datastar.NewSSE(w, r)
	tokenID := r.PathValue("id")
	err := h.tokens.Revoke(user.ID, tokenID)
	switch {
	case errors.Is(err, repository.ErrAPITokenNotFound):
		patchAPITokenToast(sse, toast.Props{Title: "Token not found", Description: "That token was already revoked.", Type: toast.TypeError})
	case err != nil:
		slog.Error("failed to revoke api token", "error", err, "user_id", user.ID, "token_id", tokenID)
		patchAPITokenToast(sse, toast.Props{Title: "Token could not be revoked", Description: "Please try again.", Type: toast.TypeError})
		return
	default:
		slog.Info("api token revoked", "token_id", tokenID, "user_id", user.ID)
		patchAPITokenToast(sse, toast.Props{Title: "Token revoked", Description: "Requests with it no longer work.", Type: toast.TypeSuccess})
	}
	view, err := apiTokensView(r, h.tokens, newAPITokenForm(r))
	if err != nil {
		slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
		return
	}
	patchAPITokens(sse, pages.APITokenList(view))
}

func patchAPITokens(sse *datastar.ServerSentEventGenerator, component templ.Component, options ...datastar.PatchElementOption) {
	err := sse.PatchElementTempl(component, options...)
	if err != nil {
		slog.Error("render Datastar elements failed", "error", err)
	}
}

func patchAPITokenToast(sse *datastar.ServerSentEventGenerator, props toast.Props) {
	patchAPITokens(sse, toast.Toast(props), datastar.WithSelector("#toast-container"), datastar.WithModeAppend())
}
