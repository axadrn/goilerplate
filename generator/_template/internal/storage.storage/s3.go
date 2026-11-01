package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Save stores a file at the given path
	Save(path string, file io.Reader) error

	// Delete removes a file at the given path
	Delete(path string) error

	// URL returns the public URL for accessing the file
	URL(path string) string
}

// S3Storage implements FileStorage for S3-compatible storage
// Works with AWS S3, Cloudflare R2, DigitalOcean Spaces, Backblaze B2, etc.
type S3Storage struct {
	client               *s3.Client
	presignClient        *s3.PresignClient
	bucket               string
	region               string
	endpoint             string        // Optional: for custom endpoints (R2, DO Spaces, etc.)
	publicURL            string        // Base URL for generating URLs
	presignExpiryPublic  time.Duration // Expiry for public files (7 days default)
	presignExpiryPrivate time.Duration // Expiry for private files (1 hour default)
}

// S3Config holds configuration for S3 storage
type S3Config struct {
	Region               string
	Bucket               string
	AccessKey            string
	SecretKey            string
	Endpoint             string        // Optional: for S3-compatible services
	PresignExpiryPublic  time.Duration // Expiry for public files
	PresignExpiryPrivate time.Duration // Expiry for private files
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	slog.Info("initializing S3 storage",
		"bucket", cfg.Bucket,
		"region", cfg.Region,
		"endpoint", cfg.Endpoint,
	)
	ctx := context.Background()

	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(cfg.Region))

	// Add static credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional custom endpoint
	var client *s3.Client
	if cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for some S3-compatible services
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	publicURL := cfg.Endpoint
	if publicURL == "" {
		// Standard AWS S3 URL
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
	} else {
		// Custom endpoint (R2, DO Spaces, etc.)
		publicURL = strings.TrimSuffix(cfg.Endpoint, "/") + "/" + cfg.Bucket
	}

	presignClient := s3.NewPresignClient(client)

	storage := &S3Storage{
		client:               client,
		presignClient:        presignClient,
		bucket:               cfg.Bucket,
		region:               cfg.Region,
		endpoint:             cfg.Endpoint,
		publicURL:            publicURL,
		presignExpiryPublic:  cfg.PresignExpiryPublic,
		presignExpiryPrivate: cfg.PresignExpiryPrivate,
	}

	// Auto-create bucket if it doesn't exist
	if err := storage.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return storage, nil
}

// ensureBucket checks if bucket exists, creates it if not
// Retries connection for up to 30s to handle slow container startup
func (s *S3Storage) ensureBucket(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		// Check if bucket exists
		_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(s.bucket),
		})
		if err == nil {
			return nil // Bucket exists
		}

		// Try to create bucket (maybe it just doesn't exist yet)
		_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.bucket),
		})
		if createErr == nil {
			slog.Info("created S3 bucket", "bucket", s.bucket)
			return nil
		}

		lastErr = err
		slog.Debug("waiting for S3 storage", "attempt", attempt, "bucket", s.bucket)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("S3 storage not available after 30s: %w", lastErr)
}

// Save stores a file in S3
func (s *S3Storage) Save(path string, file io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Delete removes a file from S3
func (s *S3Storage) Delete(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// URL returns the public URL for accessing the file
// Deprecated: Use PublicURL() or PresignedURL() directly
func (s *S3Storage) URL(path string) string {
	return fmt.Sprintf("%s/%s", s.publicURL, path)
}

// PublicURL returns a presigned URL with long expiry for public files (avatars, profile pics)
// Uses 7-day expiry by default. Stricter than GitHub (permanent) but simpler than bucket policies
func (s *S3Storage) PublicURL(path string) string {
	url, err := s.PresignedURL(path, s.presignExpiryPublic)
	if err != nil {
		// Fallback to direct URL if presigning fails
		return fmt.Sprintf("%s/%s", s.publicURL, path)
	}
	return url
}

// PresignedURL generates a presigned URL for temporary access (for private files)
func (s *S3Storage) PresignedURL(path string, expiry time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	presignedReq, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GetPresignExpiryPrivate returns the configured presign expiry for private files
func (s *S3Storage) GetPresignExpiryPrivate() time.Duration {
	return s.presignExpiryPrivate
}
