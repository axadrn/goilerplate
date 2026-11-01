package generator

// minimalConfig selects htmx on SQLite with Stripe, SMTP and no modules.
func minimalConfig() Config {
	return Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkHTMX,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
	}
}

func fullSQLiteHTMXConfig() Config {
	return Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkHTMX,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
		Workspaces:  true,
		OAuth:       []OAuth{OAuthGoogle, OAuthGitHub},
		Storage:     true,
		Content:     []Content{ContentBlog, ContentDocs},
	}
}
