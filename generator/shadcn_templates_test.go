package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var rawFrontendControl = regexp.MustCompile(`<(?:button|select|table)\b`)

func TestFrontendTemplatesUseShadcnPrimitives(t *testing.T) {
	err := filepath.WalkDir("_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".templ" {
			return nil
		}
		slashPath, _, err := splitConditions(filepath.ToSlash(path))
		if err != nil {
			return err
		}
		if !strings.Contains(slashPath, "/blocks/") && !strings.Contains(slashPath, "/layouts/") && !strings.Contains(slashPath, "/pages/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(content)
		if rawFrontendControl.MatchString(source) {
			t.Errorf("%s contains a raw button, select, or table instead of a shared shadcn-templ primitive", path)
		}
		if strings.Contains(source, "@label.Label") || strings.Contains(source, "/components/label\"") {
			t.Errorf("%s uses the standalone label instead of the shadcn-templ Field API", path)
		}
		name := filepath.Base(slashPath)
		isAuthPage := strings.Contains(slashPath, "/pages/") && strings.HasPrefix(name, "auth")
		isAuthPartial := strings.HasSuffix(name, "_form.templ")
		if isAuthPage && !isAuthPartial && !strings.Contains(source, "@blocks.AuthShell") {
			t.Errorf("%s does not use the shared shadcn-templ auth block", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBillingPricingCardsUseNativeCardFooter(t *testing.T) {
	paths, err := filepath.Glob("_template/internal/ui/pages.htmx.datastar/app_billing.*.templ")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("found %d billing templates, want 4", len(paths))
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(content)
		if !strings.Contains(source, `@card.Footer(card.FooterProps{Class: "mt-auto"})`) {
			t.Errorf("%s does not use the native card footer spacing", path)
		}
		if strings.Contains(source, `card.FooterProps{Class: "pt-0`) {
			t.Errorf("%s overrides the native card footer padding", path)
		}
		if !strings.Contains(source, "if plan.ID == currentPlanID || (plan.Purchasable && !hasProviderSubscription) {") {
			t.Errorf("%s can render an empty card footer", path)
		}
	}
}

func TestGeneratorHasNoTemplUIDependency(t *testing.T) {
	err := filepath.WalkDir("_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := strings.ToLower(string(content))
		if strings.Contains(source, "github.com/templui/") || strings.Contains(source, "templui.io") {
			t.Errorf("%s contains a legacy templUI dependency or URL", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
