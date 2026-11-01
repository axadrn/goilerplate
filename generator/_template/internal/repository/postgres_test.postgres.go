package repository_test

import (
	"os"
	"testing"

	"{{.ModulePath}}/internal/testpostgres"
)

func TestMain(m *testing.M) {
	os.Exit(testpostgres.Run(m))
}
