package testpostgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const connectionEnvironment = "GOILERPLATE_TEST_POSTGRES_URL"

func Run(m *testing.M) int {
	container, connection, err := start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL tests require Docker: %v\n", err)
		return 1
	}
	if err := os.Setenv(connectionEnvironment, connection); err != nil {
		_ = stop(container)
		fmt.Fprintf(os.Stderr, "configure PostgreSQL tests: %v\n", err)
		return 1
	}

	code := m.Run()
	if err := stop(container); err != nil {
		fmt.Fprintf(os.Stderr, "remove PostgreSQL test container: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func start() (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"docker", "run",
		"--detach",
		"--rm",
		"--publish", "127.0.0.1::5432",
		"--tmpfs", "/var/lib/postgresql/data",
		"--env", "POSTGRES_DB=postgres",
		"--env", "POSTGRES_USER=postgres",
		"--env", "POSTGRES_PASSWORD=postgres",
		"postgres:17-alpine",
	)
	var stderr strings.Builder
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", "", fmt.Errorf("start container: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	container := strings.TrimSpace(string(output))
	if container == "" {
		return "", "", fmt.Errorf("start container: Docker returned no container ID")
	}

	portOutput, err := exec.CommandContext(ctx, "docker", "port", container, "5432/tcp").CombinedOutput()
	if err != nil {
		_ = stop(container)
		return "", "", fmt.Errorf("find container port: %w: %s", err, strings.TrimSpace(string(portOutput)))
	}
	address := strings.TrimSpace(string(portOutput))
	if address == "" {
		_ = stop(container)
		return "", "", fmt.Errorf("find container port: Docker returned no address")
	}
	connection := "postgres://postgres:postgres@" + address + "/postgres?sslmode=disable"
	database, err := sql.Open("pgx", connection)
	if err != nil {
		_ = stop(container)
		return "", "", fmt.Errorf("open container database: %w", err)
	}
	defer database.Close()
	readyBy := time.Now().Add(30 * time.Second)
	for {
		err := database.PingContext(ctx)
		if err == nil {
			return container, connection, nil
		}
		if time.Now().After(readyBy) {
			_ = stop(container)
			return "", "", fmt.Errorf("PostgreSQL did not become ready: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func stop(container string) error {
	output, err := exec.Command("docker", "rm", "--force", container).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func DatabaseURL(t testing.TB) string {
	t.Helper()
	base := os.Getenv(connectionEnvironment)
	if base == "" {
		t.Fatal("PostgreSQL test container is not running")
	}
	name := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		_ = admin.Close()
		t.Fatalf("create PostgreSQL test database: %v", err)
	}
	if err := admin.Close(); err != nil {
		t.Fatalf("close PostgreSQL admin connection: %v", err)
	}

	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse PostgreSQL test connection: %v", err)
	}
	parsed.Path = "/" + name

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		admin, err := sql.Open("pgx", base)
		if err != nil {
			t.Errorf("open PostgreSQL cleanup connection: %v", err)
			return
		}
		defer admin.Close()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop PostgreSQL test database: %v", err)
		}
	})
	return parsed.String()
}
