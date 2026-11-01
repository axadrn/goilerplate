package middleware

import "net/http"

// Chain applies multiple middleware in order (first to last)
// The middleware are executed in the order they are provided
//
// Example:
//
//	handler := Chain(mux,
//	    AuthMiddleware(...),  // Executes first
//	    WithURLPath,          // Executes second
//	    Config(...),          // Executes third
//	)
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// Apply middleware in reverse order so they execute in the order provided
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
