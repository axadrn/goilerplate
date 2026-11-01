package model

import (
	"strings"
	"time"
)

type Session struct {
	ID                 string    `db:"id"`
	UserID             string    `db:"user_id"`
	ImpersonatorUserID *string   `db:"impersonator_user_id"`
	TokenHash          string    `db:"token_hash"`
	UserAgent          string    `db:"user_agent"`
	ExpiresAt          time.Time `db:"expires_at"`
	LastSeenAt         time.Time `db:"last_seen_at"`
	CreatedAt          time.Time `db:"created_at"`
}

func (s *Session) IsExpired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}

func (s *Session) IsImpersonating() bool {
	return s != nil && s.ImpersonatorUserID != nil && *s.ImpersonatorUserID != ""
}

func (s Session) DisplayName() string {
	browser := sessionBrowser(s.UserAgent)
	device := sessionDevice(s.UserAgent)
	if browser != "" && device != "" {
		return browser + " on " + device
	}
	if browser != "" {
		return browser
	}
	if device != "" {
		return "Browser on " + device
	}
	return "Browser session"
}

func sessionBrowser(userAgent string) string {
	switch {
	case strings.Contains(userAgent, "Edg/"):
		return "Edge"
	case strings.Contains(userAgent, "Chrome/") || strings.Contains(userAgent, "CriOS/"):
		return "Chrome"
	case strings.Contains(userAgent, "Firefox/") || strings.Contains(userAgent, "FxiOS/"):
		return "Firefox"
	case strings.Contains(userAgent, "Safari/"):
		return "Safari"
	default:
		return ""
	}
}

func sessionDevice(userAgent string) string {
	switch {
	case strings.Contains(userAgent, "iPhone"):
		return "iPhone"
	case strings.Contains(userAgent, "iPad"):
		return "iPad"
	case strings.Contains(userAgent, "Android"):
		return "Android"
	case strings.Contains(userAgent, "Macintosh"):
		return "macOS"
	case strings.Contains(userAgent, "Windows"):
		return "Windows"
	case strings.Contains(userAgent, "Linux"):
		return "Linux"
	default:
		return ""
	}
}
