package handler

import (
	"encoding/base64"
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/ui/pages"
)

const authEmailCookieName = "auth_email"

func setAuthEmailCookie(w http.ResponseWriter, r *http.Request, email string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authEmailCookieName,
		Value:    base64.RawURLEncoding.EncodeToString([]byte(email)),
		Path:     "/auth",
		MaxAge:   10 * 60,
		HttpOnly: true,
		Secure:   ctxkeys.RenderConfig(r.Context()).IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
}

func authEmail(r *http.Request) string {
	cookie, err := r.Cookie(authEmailCookieName)
	if err != nil {
		return ""
	}
	email, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return ""
	}
	return string(email)
}

func (h *authHandler) MagicLinkSentPage(w http.ResponseWriter, r *http.Request) {
	email := authEmail(r)
	if email == "" {
		redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	render(w, r, pages.AuthMagicLinkSent(email))
}

func (h *authHandler) ForgotPasswordSentPage(w http.ResponseWriter, r *http.Request) {
	email := authEmail(r)
	if email == "" {
		redirect(w, r, "/auth/forgot-password", http.StatusSeeOther)
		return
	}
	render(w, r, pages.AuthForgotPasswordSent(email))
}
