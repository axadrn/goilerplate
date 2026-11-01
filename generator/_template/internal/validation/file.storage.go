package validation

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// FileConstraints defines validation rules for file uploads
type FileConstraints struct {
	AllowedMimeTypes  map[string]bool
	AllowedExtensions map[string]bool
	MaxSize           int64
}

var (
	// ImageConstraints defines validation rules for image uploads
	ImageConstraints = FileConstraints{
		AllowedMimeTypes: map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/webp": true,
		},
		AllowedExtensions: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".webp": true,
		},
		MaxSize: 5 << 20, // 5MB
	}

	// DocumentConstraints defines validation rules for document uploads
	// Ready for future PDF/document support
	DocumentConstraints = FileConstraints{
		AllowedMimeTypes: map[string]bool{
			"application/pdf": true,
		},
		AllowedExtensions: map[string]bool{
			".pdf": true,
		},
		MaxSize: 10 << 20, // 10MB
	}
)

// ValidateFile validates a file upload against one or more constraint sets
// If multiple constraints are provided, file must match at least one (OR logic)
// Example: ValidateFile(header, ImageConstraints, DocumentConstraints) allows images OR PDFs
func ValidateFile(header *multipart.FileHeader, constraints ...FileConstraints) error {
	if len(constraints) == 0 {
		return fmt.Errorf("no file constraints provided")
	}

	// Try each constraint set. File must match at least one
	var lastErr error
	for _, constraint := range constraints {
		err := validateAgainstConstraint(header, constraint)
		if err == nil {
			return nil // Match found! ✅
		}
		lastErr = err
	}

	// No match found. Return last error
	return lastErr
}

// validateAgainstConstraint validates a file against a single constraint set
func validateAgainstConstraint(header *multipart.FileHeader, constraints FileConstraints) error {
	// Check file size first (before reading content)
	if header.Size > constraints.MaxSize {
		maxMB := constraints.MaxSize / (1 << 20)
		return fmt.Errorf("file too large: maximum size is %d MB", maxMB)
	}

	// Open file to read magic numbers
	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read first 512 bytes for magic number detection
	// http.DetectContentType reads max 512 bytes to determine MIME type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file pointer to beginning for later use
	seeker, ok := file.(io.Seeker)
	if ok {
		_, err = seeker.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("failed to reset file pointer: %w", err)
		}
	}

	// Detect actual content type from file content (magic numbers)
	// This cannot be faked by just changing Content-Type header
	detectedType := http.DetectContentType(buffer[:n])

	// Validate detected type against whitelist
	if !constraints.AllowedMimeTypes[detectedType] {
		return fmt.Errorf("invalid file type (detected: %s)", detectedType)
	}

	// Additional validation: check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !constraints.AllowedExtensions[ext] {
		return fmt.Errorf("invalid file extension: %s", ext)
	}

	return nil
}
