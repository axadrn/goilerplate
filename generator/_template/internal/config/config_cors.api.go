package config

import (
	"fmt"
	"net/url"
	"strings"
)

// origins reads a comma-separated list of browser origins such as
// https://app.example.com. Each entry must be a scheme, a host and an
// optional port, without path, query or wildcard, because the CORS
// middleware compares the Origin header exactly.
func (r *envReader) origins(key string) []string {
	value := r.string(key, "")
	if value == "" {
		return nil
	}

	origins := make([]string, 0)
	seen := make(map[string]struct{})
	for _, candidate := range strings.Split(value, ",") {
		candidate = strings.TrimSpace(candidate)
		origin, ok := normalizeOrigin(candidate)
		if !ok {
			r.errors = append(r.errors, fmt.Errorf("%s must contain comma-separated origins like https://app.example.com: %q", key, candidate))
			continue
		}
		_, exists := seen[origin]
		if exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}

// normalizeOrigin returns the origin in the form browsers send it: lowercase
// scheme and host, no trailing slash.
func normalizeOrigin(value string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSuffix(value, "/"))
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.Contains(parsed.Host, "*") {
		return "", false
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), true
}
