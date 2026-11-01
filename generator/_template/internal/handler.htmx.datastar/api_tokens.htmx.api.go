package handler

import (
	"errors"
	"log/slog"
	"net/http"

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

// Create stores a token and answers with the fresh dialog, the list and the
// one-time reveal. Invalid input re-renders the form inside the open dialog.
func (h *APITokensHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	input, form := parseAPITokenForm(r)
	if len(form.Errors) == 0 {
		token, raw, err := h.tokens.Create(user.ID, input)
		if err == nil {
			slog.Info("api token created", "token_id", token.ID, "user_id", user.ID)
			view, err := apiTokensView(r, h.tokens, newAPITokenForm(r))
			if err != nil {
				slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
				renderOOB(w, r, toast.Toast(toast.Props{Title: "Token created", Description: "Reload the page to see it in the list.", Type: toast.TypeInfo}), "beforeend:#toast-container")
				render(w, r, pages.APITokenReveal(raw))
				return
			}
			render(w, r, pages.APITokenCreateDialog(view))
			render(w, r, pages.APITokenList(view))
			render(w, r, pages.APITokenReveal(raw))
			return
		}
		field, message, ok := apiTokenFormError(err)
		if !ok {
			slog.Error("failed to create api token", "error", err, "user_id", user.ID)
			renderOOB(w, r, toast.Toast(toast.Props{Title: "Token could not be created", Description: "Please try again.", Type: toast.TypeError}), "beforeend:#toast-container")
			return
		}
		form.Errors[field] = message
	}
	view, err := apiTokensView(r, h.tokens, form)
	if err != nil {
		slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Token could not be created", Description: "Please try again.", Type: toast.TypeError}), "beforeend:#toast-container")
		return
	}
	render(w, r, pages.APITokenCreateForm(view, true))
}

// Revoke deletes a token and answers with the updated list.
func (h *APITokensHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	tokenID := r.PathValue("id")
	err := h.tokens.Revoke(user.ID, tokenID)
	switch {
	case errors.Is(err, repository.ErrAPITokenNotFound):
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Token not found", Description: "That token was already revoked.", Type: toast.TypeError}), "beforeend:#toast-container")
	case err != nil:
		slog.Error("failed to revoke api token", "error", err, "user_id", user.ID, "token_id", tokenID)
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Token could not be revoked", Description: "Please try again.", Type: toast.TypeError}), "beforeend:#toast-container")
		return
	default:
		slog.Info("api token revoked", "token_id", tokenID, "user_id", user.ID)
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Token revoked", Description: "Requests with it no longer work.", Type: toast.TypeSuccess}), "beforeend:#toast-container")
	}
	view, err := apiTokensView(r, h.tokens, newAPITokenForm(r))
	if err != nil {
		slog.Error("failed to list api tokens", "error", err, "user_id", user.ID)
		return
	}
	render(w, r, pages.APITokenList(view))
}
