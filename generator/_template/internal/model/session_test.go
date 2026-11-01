package model

import "testing"

func TestSessionDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{name: "safari on macOS", userAgent: "Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15", want: "Safari on macOS"},
		{name: "edge on Windows", userAgent: "Mozilla/5.0 (Windows NT 10.0) Chrome/128.0 Safari/537.36 Edg/128.0", want: "Edge on Windows"},
		{name: "unknown", userAgent: "", want: "Browser session"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := Session{UserAgent: test.userAgent}
			if got := session.DisplayName(); got != test.want {
				t.Fatalf("DisplayName() = %q, want %q", got, test.want)
			}
		})
	}
}
