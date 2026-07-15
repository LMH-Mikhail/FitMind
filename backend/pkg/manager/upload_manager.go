package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fitmind/backend/pkg/conf"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrUploadFileEmpty       = errors.New("upload file is empty")
	ErrUnsupportedUploadType = errors.New("unsupported upload file type")
	ErrInvalidUploadPath     = errors.New("invalid upload file path")
)

type UploadManager struct {
	uploadDir       string
	publicUploadURL string
	now             func() time.Time
}

type SaveUploadParams struct {
	UserID     string
	FileHeader *multipart.FileHeader
}

type UploadedFile struct {
	OriginalName string `json:"originalName"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
	RelativePath string `json:"relativePath"`
	StoragePath  string `json:"storagePath"`
	PublicURL    string `json:"publicUrl"`
}

func NewUploadManager(storage conf.StorageConfig) *UploadManager {
	uploadDir := strings.TrimSpace(storage.UploadDir)
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	absoluteUploadDir, err := filepath.Abs(uploadDir)
	if err == nil {
		uploadDir = absoluteUploadDir
	}

	publicUploadURL := strings.TrimSpace(storage.PublicUploadURL)
	if publicUploadURL == "" {
		publicUploadURL = "./uploads"
	}

	return &UploadManager{
		uploadDir:       uploadDir,
		publicUploadURL: publicUploadURL,
		now:             time.Now,
	}
}

func (manager *UploadManager) SaveClothingImage(ctx context.Context, params SaveUploadParams) (*UploadedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if params.FileHeader == nil || params.FileHeader.Size <= 0 {
		return nil, ErrUploadFileEmpty
	}

	userSegment := safePathSegment(params.UserID)
	if userSegment == "" {
		return nil, ErrInvalidUploadPath
	}

	source, err := params.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer source.Close()

	contentType, err := detectContentType(source)
	if err != nil {
		return nil, err
	}

	extension, ok := imageExtension(contentType)
	if !ok {
		return nil, ErrUnsupportedUploadType
	}

	fileName, err := randomFileName(extension)
	if err != nil {
		return nil, err
	}

	date := manager.now()
	relativeDir := filepath.Join(
		"clothing",
		userSegment,
		date.Format("2006"),
		date.Format("01"),
		date.Format("02"),
	)

	storageDir, err := manager.safeStoragePath(relativeDir)
	if err != nil {
		return nil, err
	}

	if err = os.MkdirAll(storageDir, 0755); err != nil {
		return nil, err
	}

	relativePath := filepath.Join(relativeDir, fileName)
	storagePath, err := manager.safeStoragePath(relativePath)
	if err != nil {
		return nil, err
	}

	tempPath := storagePath + ".tmp"
	target, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	_, copyErr := io.Copy(target, &contextReader{
		ctx:    ctx,
		reader: source,
	})

	closeErr := target.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return nil, closeErr
	}

	if err = os.Rename(tempPath, storagePath); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}

	return &UploadedFile{
		OriginalName: params.FileHeader.Filename,
		FileName:     fileName,
		ContentType:  contentType,
		Size:         params.FileHeader.Size,
		RelativePath: filepath.ToSlash(relativePath),
		StoragePath:  storagePath,
		PublicURL:    manager.buildPublicURL(relativePath),
	}, nil
}

func (manager *UploadManager) DeleteByRelativePath(relativePath string) error {
	storagePath, err := manager.safeStoragePath(relativePath)
	if err != nil {
		return err
	}

	err = os.Remove(storagePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (manager *UploadManager) safeStoragePath(relativePath string) (string, error) {
	cleanRelativePath := filepath.Clean(filepath.FromSlash((strings.TrimSpace(relativePath))))
	if cleanRelativePath == "." || strings.HasPrefix(cleanRelativePath, "..") || filepath.IsAbs(cleanRelativePath) {
		return "", ErrInvalidUploadPath
	}

	baseDir := filepath.Clean(manager.uploadDir)
	fullPath := filepath.Join(baseDir, cleanRelativePath)

	if fullPath != baseDir && !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		return "", ErrInvalidUploadPath
	}

	return fullPath, nil
}

func (manager *UploadManager) buildPublicURL(relativePath string) string {
	prefix := strings.TrimRight(manager.publicUploadURL, "/")
	cleanRelativePath := strings.TrimLeft(filepath.ToSlash(relativePath), "/")

	if prefix == "" {
		return "/" + cleanRelativePath
	}
	return prefix + "/" + cleanRelativePath
}

func detectContentType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return http.DetectContentType(buffer[:n]), nil
}

func imageExtension(contentType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func randomFileName(extension string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder

	for _, char := range value {
		if char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' ||
			char == '-' ||
			char == '_' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}

	return reader.reader.Read(buffer)
}

func IsUnsupportedUploadType(err error) bool {
	return errors.Is(err, ErrUnsupportedUploadType)
}

func IsInvalidUploadPath(err error) bool {
	return errors.Is(err, ErrInvalidUploadPath)
}

func IsUploadFileEmpty(err error) bool {
	return errors.Is(err, ErrUploadFileEmpty)
}
