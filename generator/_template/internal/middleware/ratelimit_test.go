package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestRateLimiterEnforcesLimitPerIP(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	t.Cleanup(limiter.Close)
	if !limiter.Allow("192.0.2.1") || !limiter.Allow("192.0.2.1") {
		t.Fatal("RateLimiter rejected a request below the limit")
	}
	if limiter.Allow("192.0.2.1") {
		t.Fatal("RateLimiter accepted a request above the limit")
	}
	if !limiter.Allow("192.0.2.2") {
		t.Fatal("RateLimiter leaked limits across IP addresses")
	}
}

func TestAuthEmailRateLimitIsScopedByIPAndNormalizedRecipient(t *testing.T) {
	ipLimiter := NewRateLimiter(3, time.Hour)
	recipientLimiter := NewRateLimiter(2, time.Hour)
	t.Cleanup(ipLimiter.Close)
	t.Cleanup(recipientLimiter.Close)
	handler := rateLimitAuthEmail(NewClientIPResolver(nil), ipLimiter, recipientLimiter)(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := func(ip, email string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader("email="+url.QueryEscape(email)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = ip + ":1234"
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		return recorder
	}

	if recorder := request("192.0.2.1", "Reader@Example.com"); recorder.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d", recorder.Code)
	}
	if recorder := request("192.0.2.2", " reader@example.com "); recorder.Code != http.StatusNoContent {
		t.Fatalf("second request status = %d", recorder.Code)
	}
	if recorder := request("192.0.2.3", "READER@example.com"); recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "3600" {
		t.Fatalf("recipient-limited status = %d, Retry-After = %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}
	if recorder := request("192.0.2.1", "other-1@example.com"); recorder.Code != http.StatusNoContent {
		t.Fatalf("same IP second recipient status = %d", recorder.Code)
	}
	if recorder := request("192.0.2.1", "other-2@example.com"); recorder.Code != http.StatusNoContent {
		t.Fatalf("same IP third recipient status = %d", recorder.Code)
	}
	if recorder := request("192.0.2.1", "other-3@example.com"); recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("IP-limited status = %d", recorder.Code)
	}
}

func TestEmailChangeRateLimitIsScopedByUser(t *testing.T) {
	userLimiter := NewRateLimiter(2, time.Hour)
	recipientLimiter := NewRateLimiter(2, time.Hour)
	t.Cleanup(userLimiter.Close)
	t.Cleanup(recipientLimiter.Close)
	handler := rateLimitUserEmailChanges(userLimiter, recipientLimiter)(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := func(userID, email string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/app/workspace/account/email", strings.NewReader("email="+url.QueryEscape(email)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = req.WithContext(ctxkeys.WithUser(req.Context(), &model.User{ID: userID}))
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		return recorder
	}

	if request("user-1", "first@example.com").Code != http.StatusNoContent || request("user-1", "second@example.com").Code != http.StatusNoContent {
		t.Fatal("email change limiter rejected a request below the limit")
	}
	if recorder := request("user-1", "third@example.com"); recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "3600" {
		t.Fatalf("limited status = %d, Retry-After = %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}
	if recorder := request("user-2", "victim@example.com"); recorder.Code != http.StatusNoContent {
		t.Fatalf("different user status = %d", recorder.Code)
	}
	if recorder := request("user-3", " VICTIM@example.com "); recorder.Code != http.StatusNoContent {
		t.Fatalf("second recipient request status = %d", recorder.Code)
	}
	if recorder := request("user-4", "victim@example.com"); recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("recipient-limited status = %d", recorder.Code)
	}
}

func TestGetClientIPIgnoresForwardedHeadersFromUntrustedClient(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("X-Real-IP", "192.0.2.20")
	request.Header.Set("X-Forwarded-For", "192.0.2.30, 192.0.2.40")

	if got := getClientIPWithTrustedProxies(request, trustedProxyCIDRs("10.0.0.0/8")); got != "192.0.2.10" {
		t.Fatalf("getClientIP() = %q, want 192.0.2.10", got)
	}
}

func TestGetClientIPUsesForwardedAddressFromLoopbackProxy(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.99, 192.0.2.30")

	if got := getClientIPWithTrustedProxies(request, trustedProxyCIDRs("127.0.0.0/8")); got != "192.0.2.30" {
		t.Fatalf("getClientIP() = %q, want 192.0.2.30", got)
	}
}

func TestGetClientIPUsesForwardedAddressFromTrustedNetworkProxy(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.20.30.40:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.99, 192.0.2.30")

	if got := getClientIPWithTrustedProxies(request, trustedProxyCIDRs("10.0.0.0/8")); got != "192.0.2.30" {
		t.Fatalf("getClientIP() = %q, want 192.0.2.30", got)
	}
}

func TestGetClientIPSkipsTrustedAddressesInForwardedChain(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.20.30.40:1234"
	request.Header.Set("X-Forwarded-For", "192.0.2.30, 10.10.0.5")

	if got := getClientIPWithTrustedProxies(request, trustedProxyCIDRs("10.0.0.0/8")); got != "192.0.2.30" {
		t.Fatalf("getClientIP() = %q, want 192.0.2.30", got)
	}
}

func TestClientIPResolverUsesCopiedTrustedProxies(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.20.30.40:1234"
	request.Header.Set("X-Forwarded-For", "192.0.2.30")
	trusted := trustedProxyCIDRs("10.0.0.0/8")
	resolver := NewClientIPResolver(trusted)
	trusted[0] = netip.MustParsePrefix("172.16.0.0/12")

	got := resolver.ClientIP(request)
	if got != "192.0.2.30" {
		t.Fatalf("ClientIP() = %q, want 192.0.2.30", got)
	}
}

func trustedProxyCIDRs(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}
