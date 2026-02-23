package utils

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ahmadqo/digital-achievement-ledger/internal/config"
)

type StorageService struct {
	client   *minio.Client
	bucket   string
	endpoint string
	useSSL   bool
}

type UploadResult struct {
	FileURL  string
	FileName string
	FileSize int64
}

// Allowed file types untuk attachment
var AllowedAttachmentTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"application/pdf": ".pdf",
}

const MaxFileSize = 10 * 1024 * 1024 // 10 MB

func NewStorageService(cfg *config.MinIOConfig) (*StorageService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.User, cfg.Password, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Pastikan bucket ada
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	return &StorageService{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: fmt.Sprintf("%s://%s", scheme, cfg.Endpoint),
		useSSL:   cfg.UseSSL,
	}, nil
}

// UploadFile upload file ke MinIO dan kembalikan URL-nya
func (s *StorageService) UploadFile(ctx context.Context, folder string, data []byte, contentType string) (*UploadResult, error) {
	// Validasi content type
	ext, ok := AllowedAttachmentTypes[contentType]
	if !ok {
		return nil, fmt.Errorf("tipe file tidak diizinkan: %s", contentType)
	}

	// Validasi ukuran
	if len(data) > MaxFileSize {
		return nil, fmt.Errorf("ukuran file melebihi batas maksimal 10MB")
	}

	// Generate nama file unik
	fileName := fmt.Sprintf("%s/%s-%s%s",
		folder,
		time.Now().Format("20060102"),
		uuid.New().String()[:8],
		ext,
	)

	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, fileName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal upload file: %w", err)
	}

	fileURL := fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, fileName)

	return &UploadResult{
		FileURL:  fileURL,
		FileName: filepath.Base(fileName),
		FileSize: int64(len(data)),
	}, nil
}

// UploadPDF upload file PDF ke MinIO (untuk sertifikat)
func (s *StorageService) UploadPDF(ctx context.Context, folder string, data []byte, name string) (string, error) {
	// Sanitasi nama file
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	fileName := fmt.Sprintf("%s/%s-%s.pdf", folder, name, uuid.New().String()[:8])

	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, fileName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/pdf",
	})
	if err != nil {
		return "", fmt.Errorf("gagal upload PDF: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, fileName), nil
}

// DeleteFile hapus file dari MinIO
func (s *StorageService) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract object name dari URL
	prefix := fmt.Sprintf("%s/%s/", s.endpoint, s.bucket)
	objectName := strings.TrimPrefix(fileURL, prefix)

	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}