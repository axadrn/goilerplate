package middleware

import (
	"errors"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/service"
)

var errInvalidBearer = errors.New("invalid bearer token")

// authenticateBearer authenticates the session token in the Authorization
// header. It returns the session and the raw token.
func authenticateBearer(r *http.Request, authService *service.AuthService) (*model.Session, string, error) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return nil, "", errInvalidBearer
	}
	// Personal API tokens are not sessions. APITokenAuth handles them.
	if strings.HasPrefix(token, service.APITokenMarker) {
		return nil, "", errInvalidBearer
	}
	session, err := authService.AuthenticateSession(token)
	if err != nil {
		return nil, "", err
	}
	return session, token, nil
}

// hasBearerAuthorization reports whether the request sends an Authorization
// header with the Bearer scheme. Such a request authenticates with that header
// alone: AuthMiddleware never reads its cookies, even when the token is
// invalid, so CSRFProtection may skip it. Any other scheme, such as Basic auth
// added by a proxy, leaves cookie sessions and the CSRF check in place.
func hasBearerAuthorization(r *http.Request) bool {
	scheme, _, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	return strings.EqualFold(scheme, "Bearer")
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" value.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}
