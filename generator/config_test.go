package generator

import (
	"strings"
	"testing"
)

func TestConfigValidatesHeadlessAndAPI(t *testing.T) {
	headless := fullSQLiteHTMXConfig()
	headless.Framework = FrameworkHeadless
	headless.API = true
	headless.Content = nil

	tests := []struct {
		name      string
		config    func() Config
		wantError string
	}{
		{name: "minimal api htmx", config: func() Config {
			config := minimalConfig()
			config.API = true
			return config
		}},
		{name: "minimal api datastar", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkDatastar
			config.API = true
			return config
		}},
		{name: "minimal headless", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkHeadless
			config.API = true
			return config
		}},
		{name: "full headless", config: func() Config { return headless }},
		{name: "full api datastar with content", config: func() Config {
			config := fullSQLiteHTMXConfig()
			config.Framework = FrameworkDatastar
			config.API = true
			return config
		}},
		{name: "minimal headless without api", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkHeadless
			return config
		}, wantError: "Headless requires the JSON API"},
		{name: "full headless without api", config: func() Config {
			config := headless
			config.API = false
			return config
		}, wantError: "Headless requires the JSON API"},
		{name: "full headless with blog", config: func() Config {
			config := headless
			config.Content = []Content{ContentBlog}
			return config
		}, wantError: "Headless does not support blog or docs"},
		{name: "full headless with docs", config: func() Config {
			config := headless
			config.Content = []Content{ContentDocs}
			return config
		}, wantError: "Headless does not support blog or docs"},
		{name: "minimal svelte", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkSvelte
			config.API = true
			return config
		}},
		{name: "full svelte with mcp", config: func() Config {
			config := headless
			config.Framework = FrameworkSvelte
			config.MCP = true
			return config
		}},
		{name: "minimal svelte without api", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkSvelte
			return config
		}, wantError: "SvelteKit requires the JSON API"},
		{name: "full svelte with docs", config: func() Config {
			config := headless
			config.Framework = FrameworkSvelte
			config.Content = []Content{ContentDocs}
			return config
		}, wantError: "SvelteKit does not support blog or docs"},
		{name: "unknown framework", config: func() Config {
			config := minimalConfig()
			config.Framework = "none"
			return config
		}, wantError: `unsupported framework "none"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := test.config()
			normalized, err := config.normalized()
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("normalized() error = %v", err)
				}
				if normalized.IsHeadless() != (config.Framework == FrameworkHeadless) {
					t.Fatalf("IsHeadless() = %t for framework %q", normalized.IsHeadless(), config.Framework)
				}
				if normalized.IsSvelte() != (config.Framework == FrameworkSvelte) {
					t.Fatalf("IsSvelte() = %t for framework %q", normalized.IsSvelte(), config.Framework)
				}
				htmlFrontend := config.Framework == FrameworkHTMX || config.Framework == FrameworkDatastar
				if normalized.HasHTMLFrontend() != htmlFrontend {
					t.Fatalf("HasHTMLFrontend() = %t for framework %q", normalized.HasHTMLFrontend(), config.Framework)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("normalized() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestConfigFactsMapAPI(t *testing.T) {
	config := minimalConfig()
	got := facts(config)["api"]
	if got != "no_api" {
		t.Fatalf("facts api = %q, want no_api", got)
	}
	config.API = true
	got = facts(config)["api"]
	if got != "api" {
		t.Fatalf("facts api = %q, want api", got)
	}
	config.Framework = FrameworkHeadless
	got = facts(config)["framework"]
	if got != "headless" {
		t.Fatalf("facts framework = %q, want headless", got)
	}
}

func TestConfigValidatesMCP(t *testing.T) {
	headless := fullSQLiteHTMXConfig()
	headless.Framework = FrameworkHeadless
	headless.API = true
	headless.Content = nil

	tests := []struct {
		name      string
		config    func() Config
		wantError string
	}{
		{name: "minimal htmx with api and mcp", config: func() Config {
			config := minimalConfig()
			config.API = true
			config.MCP = true
			return config
		}},
		{name: "minimal headless with mcp", config: func() Config {
			config := minimalConfig()
			config.Framework = FrameworkHeadless
			config.API = true
			config.MCP = true
			return config
		}},
		{name: "full datastar with api and mcp", config: func() Config {
			config := fullSQLiteHTMXConfig()
			config.Framework = FrameworkDatastar
			config.API = true
			config.MCP = true
			return config
		}},
		{name: "full headless with mcp", config: func() Config {
			config := headless
			config.MCP = true
			return config
		}},
		{name: "minimal htmx with mcp and no api", config: func() Config {
			config := minimalConfig()
			config.MCP = true
			return config
		}, wantError: "MCP requires the JSON API"},
		{name: "full datastar with mcp and no api", config: func() Config {
			config := fullSQLiteHTMXConfig()
			config.Framework = FrameworkDatastar
			config.MCP = true
			return config
		}, wantError: "MCP requires the JSON API"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := test.config()
			normalized, err := config.normalized()
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("normalized() error = %v", err)
				}
				if !normalized.MCP {
					t.Fatal("normalized() dropped MCP")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("normalized() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestConfigFactsMapMCP(t *testing.T) {
	config := minimalConfig()
	got := facts(config)["mcp"]
	if got != "no_mcp" {
		t.Fatalf("facts mcp = %q, want no_mcp", got)
	}
	config.API = true
	config.MCP = true
	got = facts(config)["mcp"]
	if got != "mcp" {
		t.Fatalf("facts mcp = %q, want mcp", got)
	}
}

func TestConfigRequiresAPaymentProvider(t *testing.T) {
	for payment, want := range map[Payment]string{
		"":     "a payment provider is required",
		"none": `unsupported payment provider "none"`,
	} {
		config := minimalConfig()
		config.Payment = payment
		_, err := config.normalized()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("payment %q error = %v, want containing %q", payment, err, want)
		}
	}
}
