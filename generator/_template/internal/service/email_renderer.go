package service

import (
	"html/template"
	"strings"
)

type emailContent struct {
	Subject     string
	Preheader   string
	Heading     string
	Greeting    string
	Intro       []string
	ActionLabel string
	ActionURL   string
	Outro       []string
	AppName     string
}

type renderedEmail struct {
	Text string
	HTML string
}

var transactionalEmailTemplate = template.Must(template.New("transactional_email").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Subject}}</title>
</head>
<body style="margin:0;background:#f5f5f5;color:#171717;font-family:Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;">{{.Preheader}}</div>
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f5f5f5;">
    <tr>
      <td align="center" style="padding:32px 16px;">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #e5e5e5;border-radius:12px;">
          <tr>
            <td style="padding:40px 32px;">
              <h1 style="margin:0 0 24px;font-size:24px;line-height:1.25;">{{.Heading}}</h1>
              {{if .Greeting}}<p style="margin:0 0 16px;font-size:16px;line-height:1.6;">{{.Greeting}}</p>{{end}}
              {{range .Intro}}<p style="margin:0 0 16px;font-size:16px;line-height:1.6;">{{.}}</p>{{end}}
              {{if .ActionURL}}
              <p style="margin:24px 0;">
                <a href="{{.ActionURL}}" style="display:inline-block;background:#171717;color:#ffffff;text-decoration:none;font-size:16px;font-weight:600;line-height:1;padding:14px 20px;border-radius:8px;">{{.ActionLabel}}</a>
              </p>
              <p style="margin:0 0 24px;color:#737373;font-size:13px;line-height:1.5;word-break:break-all;">
                If the button does not work, copy and paste this link into your browser:<br>
                <a href="{{.ActionURL}}" style="color:#525252;">{{.ActionURL}}</a>
              </p>
              {{end}}
              {{range .Outro}}<p style="margin:0 0 16px;font-size:16px;line-height:1.6;">{{.}}</p>{{end}}
              <p style="margin:24px 0 0;font-size:16px;line-height:1.6;">Best,<br>The {{.AppName}} Team</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

func renderEmail(content emailContent) (renderedEmail, error) {
	blocks := make([]string, 0, len(content.Intro)+len(content.Outro)+4)
	blocks = appendEmailBlock(blocks, content.Heading)
	blocks = appendEmailBlock(blocks, content.Greeting)
	for _, paragraph := range content.Intro {
		blocks = appendEmailBlock(blocks, paragraph)
	}
	if content.ActionURL != "" {
		blocks = append(blocks, content.ActionLabel+": "+content.ActionURL)
	}
	for _, paragraph := range content.Outro {
		blocks = appendEmailBlock(blocks, paragraph)
	}
	blocks = append(blocks, "Best,\nThe "+content.AppName+" Team")

	var html strings.Builder
	if err := transactionalEmailTemplate.Execute(&html, content); err != nil {
		return renderedEmail{}, err
	}

	return renderedEmail{
		Text: strings.Join(blocks, "\n\n"),
		HTML: html.String(),
	}, nil
}

func appendEmailBlock(blocks []string, block string) []string {
	if strings.TrimSpace(block) == "" {
		return blocks
	}
	return append(blocks, block)
}
