package doctor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/update"
)

type Level int

const maxEnvironmentSize = 1 << 20

const (
	LevelOK Level = iota
	LevelWarning
	LevelError
)

type Check struct {
	Name    string
	Message string
	Level   Level
}

type Report struct {
	Checks []Check
	Errors int
}

type Inspector struct {
	LookPath func(string) (string, error)
	Output   func(context.Context, string, ...string) (string, error)
}

func Inspect(ctx context.Context, directory string) Report {
	inspector := Inspector{
		LookPath: exec.LookPath,
		Output: func(ctx context.Context, name string, arguments ...string) (string, error) {
			output, err := exec.CommandContext(ctx, name, arguments...).CombinedOutput()
			return strings.TrimSpace(string(output)), err
		},
	}
	return inspector.Inspect(ctx, directory)
}

func (i Inspector) Inspect(ctx context.Context, directory string) Report {
	var report Report
	root, err := findProjectRoot(directory)
	if err != nil {
		return report.add("project", err.Error(), LevelError)
	}
	lock, err := update.ReadLock(root)
	if err != nil {
		return report.add("project", err.Error(), LevelError)
	}
	report = report.add("project", lock.TemplateVersion+" "+lock.Answers.Edition, LevelOK)

	module, requiredGo, err := readModule(filepath.Join(root, "go.mod"))
	if err != nil {
		report = report.add("go.mod", err.Error(), LevelError)
	} else {
		if module != lock.Answers.ModulePath {
			report = report.add("go.mod", fmt.Sprintf("module is %s, lock expects %s", module, lock.Answers.ModulePath), LevelError)
		} else {
			report = report.add("go.mod", module, LevelOK)
		}
		if requiredGo != "" {
			report = i.checkVersion(ctx, report, "go", []string{"env", "GOVERSION"}, requiredGo, "Install Go "+requiredGo+" or newer")
		}
	}

	report = i.checkVersion(ctx, report, "git", []string{"--version"}, "2.38.0", "Install Git 2.38 or newer for goilerplate update")
	report = i.checkTool(report, "task", "Optional. Install Task from https://taskfile.dev/installation/", LevelWarning)
	report = i.checkTool(report, "tailwindcss", "Optional. Install the Tailwind CSS CLI from https://tailwindcss.com/docs/installation/tailwind-cli", LevelWarning)

	environmentPath := filepath.Join(root, ".env")
	if _, err := os.Stat(environmentPath); err == nil {
		environment, readErr := readEnvironment(environmentPath)
		if readErr != nil {
			report = report.add(".env", readErr.Error(), LevelError)
		} else {
			missing := missingEnvironmentKeys(lock.Answers, environment)
			if len(missing) > 0 {
				report = report.add(".env", "missing required values: "+strings.Join(missing, ", "), LevelError)
			} else {
				report = report.add(".env", "matches the selected stack", LevelOK)
			}
		}
	} else if errors.Is(err, os.ErrNotExist) {
		report = report.add(".env", "missing. Create .env from .env.example", LevelWarning)
	} else {
		report = report.add(".env", err.Error(), LevelWarning)
	}
	if lock.Answers.Database == "postgres" || lock.Answers.Mail == "smtp" {
		report = i.checkTool(report, "docker", "Optional. Install Docker or provide the selected services yourself", LevelWarning)
	}
	return report
}

func readEnvironment(name string) (map[string]string, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxEnvironmentSize {
		return nil, fmt.Errorf(".env is larger than %d bytes", maxEnvironmentSize)
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(io.LimitReader(file, maxEnvironmentSize+1))
	scanner.Buffer(make([]byte, 64*1024), maxEnvironmentSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		values[key] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func missingEnvironmentKeys(answers api.GenerationAnswers, values map[string]string) []string {
	required := []string{"APP_NAME", "APP_ENV", "APP_URL", "DB_CONNECTION", "EMAIL_FROM", "SESSION_EXPIRY"}
	if answers.Mail == "smtp" {
		required = append(required, "SMTP_HOST", "SMTP_PORT", "SMTP_TLS")
	}
	if answers.Mail == "resend" {
		required = append(required, "RESEND_API_KEY")
	}
	switch answers.Payment {
	case "stripe":
		required = append(required, "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PRICE_ID_PRO_MONTHLY", "STRIPE_PRICE_ID_PRO_YEARLY", "STRIPE_PRICE_ID_ENTERPRISE_MONTHLY", "STRIPE_PRICE_ID_ENTERPRISE_YEARLY")
	case "polar":
		required = append(required, "POLAR_API_KEY", "POLAR_WEBHOOK_SECRET", "POLAR_PRODUCT_ID_PRO_MONTHLY", "POLAR_PRODUCT_ID_PRO_YEARLY", "POLAR_PRODUCT_ID_ENTERPRISE_MONTHLY", "POLAR_PRODUCT_ID_ENTERPRISE_YEARLY")
	case "lemon_squeezy":
		required = append(required, "LEMON_SQUEEZY_API_KEY", "LEMON_SQUEEZY_WEBHOOK_SECRET", "LEMON_SQUEEZY_STORE_ID", "LEMON_SQUEEZY_VARIANT_ID_PRO_MONTHLY", "LEMON_SQUEEZY_VARIANT_ID_PRO_YEARLY", "LEMON_SQUEEZY_VARIANT_ID_ENTERPRISE_MONTHLY", "LEMON_SQUEEZY_VARIANT_ID_ENTERPRISE_YEARLY")
	}
	for _, provider := range answers.OAuth {
		switch provider {
		case "google":
			required = append(required, "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET")
		case "github":
			required = append(required, "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET")
		}
	}
	if answers.Storage {
		required = append(required, "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_ENDPOINT")
	}
	if answers.Example {
		required = append(required, "SEED_PASSWORD")
	}
	var missing []string
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func (r Report) add(name, message string, level Level) Report {
	r.Checks = append(r.Checks, Check{Name: name, Message: message, Level: level})
	if level == LevelError {
		r.Errors++
	}
	return r
}

func (i Inspector) checkTool(report Report, name, fix string, missingLevel Level) Report {
	path, err := i.LookPath(name)
	if err != nil {
		return report.add(name, fix, missingLevel)
	}
	return report.add(name, path, LevelOK)
}

func (i Inspector) checkVersion(ctx context.Context, report Report, name string, arguments []string, minimum, fix string) Report {
	if _, err := i.LookPath(name); err != nil {
		return report.add(name, fix, LevelError)
	}
	output, err := i.Output(ctx, name, arguments...)
	if err != nil {
		return report.add(name, output, LevelError)
	}
	version, ok := firstVersion(output)
	if !ok {
		return report.add(name, "could not read version from "+output, LevelError)
	}
	if compareVersions(version, minimum) < 0 {
		return report.add(name, version+" is too old. "+fix, LevelError)
	}
	return report.add(name, version, LevelOK)
}

func findProjectRoot(directory string) (string, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "goilerplate.lock")); err == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", errors.New("goilerplate.lock was not found in this directory or its parents")
		}
		root = parent
	}
}

func readModule(name string) (string, string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", "", fmt.Errorf("open go.mod: %w", err)
	}
	defer file.Close()
	var module, goVersion string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "//")
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			module = fields[1]
		}
		if len(fields) == 2 && fields[0] == "go" {
			goVersion = fields[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if module == "" {
		return "", "", errors.New("go.mod must contain a module directive")
	}
	return module, goVersion, nil
}

func firstVersion(value string) (string, bool) {
	for _, field := range strings.Fields(value) {
		candidate := strings.TrimLeftFunc(field, func(character rune) bool {
			return character < '0' || character > '9'
		})
		if _, count := numericVersion(candidate); count >= 2 {
			return candidate, true
		}
	}
	return "", false
}

func compareVersions(left, right string) int {
	leftParts, _ := numericVersion(left)
	rightParts, _ := numericVersion(right)
	for index := 0; index < 3; index++ {
		if leftParts[index] < rightParts[index] {
			return -1
		}
		if leftParts[index] > rightParts[index] {
			return 1
		}
	}
	return 0
}

func numericVersion(value string) ([3]int, int) {
	var result [3]int
	count := 0
	for _, part := range strings.Split(value, ".") {
		if count == len(result) {
			break
		}
		digits := part
		if end := strings.IndexFunc(part, func(character rune) bool {
			return character < '0' || character > '9'
		}); end >= 0 {
			digits = part[:end]
		}
		if digits == "" {
			break
		}
		number, err := strconv.Atoi(digits)
		if err != nil {
			break
		}
		result[count] = number
		count++
	}
	return result, count
}
