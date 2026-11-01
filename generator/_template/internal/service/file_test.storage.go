package service

import (
	"errors"
	"testing"
	"time"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestFileServiceDeleteRequiresOwner(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "file-owner"})
	attacker := testutil.SeedUser(t, database, testutil.UserOptions{ID: "file-attacker"})
	files := repository.NewFileRepository(database)
	file := &model.File{ID: "owned-file", UserID: owner.User.ID, OwnerType: "user", OwnerID: owner.User.ID, Type: "document", Filename: "file.txt", OriginalName: "file.txt", MimeType: "text/plain", Size: 4, StoragePath: "private/documents/file.txt", CreatedAt: time.Now()}
	if err := files.Create(file); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	service := NewFileService(files, testutil.NewMemoryStorage())
	if err := service.Delete(attacker.User.ID, file.ID); !errors.Is(err, ErrFilePermission) {
		t.Fatalf("attacker Delete() error = %v, want ErrFilePermission", err)
	}
	if _, err := files.ByID(file.ID); err != nil {
		t.Fatalf("file was deleted by non-owner: %v", err)
	}
	if err := service.Delete(owner.User.ID, file.ID); err != nil {
		t.Fatalf("owner Delete() error = %v", err)
	}
	if _, err := files.ByID(file.ID); !errors.Is(err, repository.ErrFileNotFound) {
		t.Fatalf("deleted file lookup error = %v, want ErrFileNotFound", err)
	}
}
