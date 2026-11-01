package service

import (
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"time"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/storage"
	"github.com/google/uuid"
)

var ErrFilePermission = errors.New("file does not belong to this user")

type FileService struct {
	fileRepo repository.FileRepository
	storage  storage.Storage
}

func NewFileService(fileRepo repository.FileRepository, storage storage.Storage) *FileService {
	return &FileService{
		fileRepo: fileRepo,
		storage:  storage,
	}
}

// Upload uploads a file and creates a database record
// Note: File validation (type, size, content) should be done by the caller before calling Upload
func (s *FileService) Upload(userID, ownerType, ownerID, fileType string, file multipart.File, header *multipart.FileHeader, isPublic bool) (*model.File, error) {
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Generate storage path with public/private prefix
	prefix := "private"
	if isPublic {
		prefix = "public"
	}
	folderName := fileType + "s" // avatar -> avatars
	storagePath := filepath.Join(prefix, folderName, filename)

	// Save file to storage
	err := s.storage.Save(storagePath, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create database record
	fileModel := &model.File{
		ID:           uuid.New().String(),
		UserID:       userID,
		OwnerType:    ownerType,
		OwnerID:      ownerID,
		Type:         fileType,
		Filename:     filename,
		OriginalName: header.Filename,
		MimeType:     header.Header.Get("Content-Type"),
		Size:         header.Size,
		StoragePath:  storagePath,
		Public:       isPublic,
		CreatedAt:    time.Now(),
	}

	err = s.fileRepo.Create(fileModel)
	if err != nil {
		// If DB insert fails, try to cleanup the uploaded file
		delErr := s.storage.Delete(storagePath)
		if delErr != nil {
			slog.Error("failed to delete file from storage during cleanup", "error", delErr, "path", storagePath)
		}
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return fileModel, nil
}

// Avatar retrieves the avatar for an owner (user, etc.)
func (s *FileService) Avatar(ownerType, ownerID string) (*model.File, error) {
	return s.fileRepo.FileByType(ownerType, ownerID, model.FileTypeAvatar)
}

// URL returns the appropriate URL for a file (public or presigned)
func (s *FileService) URL(file *model.File) string {
	if file == nil {
		return ""
	}

	// Type assert to check if S3 storage
	s3Storage, ok := s.storage.(*storage.S3Storage)
	if ok {
		if file.Public {
			// Public files: presigned URL with long expiry (7 days)
			return s3Storage.PublicURL(file.StoragePath)
		}
		// Private files: presigned URL with short expiry (1 hour)
		url, err := s3Storage.PresignedURL(file.StoragePath, s3Storage.GetPresignExpiryPrivate())
		if err != nil {
			// Fallback to public URL if presigning fails
			return s3Storage.PublicURL(file.StoragePath)
		}
		return url
	}

	// Local storage or other: use default URL method
	return s.storage.URL(file.StoragePath)
}

// Delete removes a file from storage and database
func (s *FileService) Delete(userID, fileID string) error {
	// Get file record
	file, err := s.fileRepo.ByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	if file.UserID != userID {
		return ErrFilePermission
	}

	// Delete from storage (best effort)
	delErr := s.storage.Delete(file.StoragePath)
	if delErr != nil {
		slog.Error("failed to delete file from storage", "error", delErr, "path", file.StoragePath)
	}

	// Delete from database
	err = s.fileRepo.Delete(fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	return nil
}

// DeleteUserAvatar deletes the user's avatar
func (s *FileService) DeleteUserAvatar(userID string) error {
	file, err := s.Avatar("user", userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return nil // No avatar to delete
		}
		return err
	}

	return s.Delete(userID, file.ID)
}

// AllUserFiles retrieves all files owned by a user (regardless of owner_type)
func (s *FileService) AllUserFiles(userID string) ([]*model.File, error) {
	return s.fileRepo.AllUserFiles(userID)
}

func (s *FileService) DeleteAllUserFilesFromStorage(userID string) error {
	files, err := s.fileRepo.AllUserFiles(userID)
	if err != nil {
		return fmt.Errorf("failed to get user files: %w", err)
	}
	return s.DeleteFilesFromStorage(files)
}

func (s *FileService) DeleteFilesFromStorage(files []*model.File) error {
	for _, file := range files {
		err := s.storage.Delete(file.StoragePath)
		if err != nil {
			slog.Warn("failed to delete file from storage", "storage_path", file.StoragePath, "error", err)
		}
	}
	return nil
}
