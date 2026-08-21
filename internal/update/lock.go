package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/axadrn/goilerplate/v3/api"
)

const maxLockSize = 64 << 10

func ReadLock(root string) (api.ProjectLock, error) {
	file, err := os.Open(filepath.Join(root, "goilerplate.lock"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return api.ProjectLock{}, errors.New("goilerplate.lock was not found")
		}
		return api.ProjectLock{}, fmt.Errorf("open goilerplate.lock: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return api.ProjectLock{}, fmt.Errorf("read goilerplate.lock size: %w", err)
	}
	if info.Size() > maxLockSize {
		return api.ProjectLock{}, errors.New("goilerplate.lock is too large")
	}

	decoder := json.NewDecoder(io.LimitReader(file, maxLockSize))
	decoder.DisallowUnknownFields()
	var lock api.ProjectLock
	if err := decoder.Decode(&lock); err != nil {
		return api.ProjectLock{}, fmt.Errorf("read goilerplate.lock: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.ProjectLock{}, errors.New("goilerplate.lock must contain one JSON object")
	}
	if lock.SchemaVersion != api.LockSchemaVersion {
		return api.ProjectLock{}, fmt.Errorf("goilerplate.lock uses unsupported schema version %d", lock.SchemaVersion)
	}
	if strings.TrimSpace(lock.TemplateVersion) == "" {
		return api.ProjectLock{}, errors.New("goilerplate.lock contains no template version")
	}
	return lock, nil
}
