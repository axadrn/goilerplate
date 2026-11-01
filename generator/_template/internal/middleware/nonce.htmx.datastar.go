package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/a-h/templ"
)

// nonceKey is the context key for storing the generated nonce
// We use a separate key from templ's internal key so we can also access
// the nonce in SecurityHeaders middleware to inject into CSP header
type nonceKey struct{}

// NonceMiddleware generates a cryptographically secure random nonce
// for each request and stores it in the context for use in:
//  1. Templ templates via templ.GetNonce(ctx)
//  2. SecurityHeaders middleware for CSP header injection
//
// The nonce is used in Content-Security-Policy to allow specific inline
// scripts while blocking all other inline JavaScript (XSS protection).
//
// How it works:
//  1. Generate 16-byte random nonce (base64 encoded = 24 chars)
//  2. Store in context via templ.WithNonce() for template access
//  3. Store in custom context key for middleware access
//  4. Templates render: <script nonce="abc123">...</script>
//  5. SecurityHeaders injects: script-src 'self' 'nonce-abc123'
//  6. Browser only executes scripts with matching nonce
func NonceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate cryptographically secure random nonce
		nonce, err := generateNonce()
		if err != nil {
			// Fallback: continue without nonce (degraded security but app still works)
			// In production this should be logged/monitored
			next.ServeHTTP(w, r)
			return
		}

		// Store nonce in context via templ's built-in method
		// This makes it available to templates via templ.GetNonce(ctx)
		ctx := templ.WithNonce(r.Context(), nonce)

		// Also store in our custom context key so SecurityHeaders can access it
		ctx = context.WithValue(ctx, nonceKey{}, nonce)

		// Continue with nonce in context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetNonce retrieves the nonce from context for use in middleware
// (templates should use templ.GetNonce() instead)
func GetNonce(ctx context.Context) string {
	nonce, _ := ctx.Value(nonceKey{}).(string)
	return nonce
}

// generateNonce creates a cryptographically secure random nonce
// Returns a base64-encoded string of 16 random bytes (24 chars output)
func generateNonce() (string, error) {
	// 16 bytes = 128 bits of entropy (sufficient for CSP nonce)
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// Base64 encode: 16 bytes → 24 characters
	return base64.StdEncoding.EncodeToString(b), nil
}
