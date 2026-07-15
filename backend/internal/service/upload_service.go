package service

import (
	"context"
	"errors"
	"fitmind/backend/pkg/header/dto"
	"fitmind/backend/pkg/manager"
	"mime/multipart"
	"strings"
)

type UploadErrorKind string

const (
	UploadKindInvalidInput    UploadErrorKind = "invalid_input"
	UploadKindUnauthorized    UploadErrorKind = "unauthorized"
	UploadKindUnsupportedType UploadErrorKind = "unsupported_type"
	UploadKindTooLarge        UploadErrorKind = "too_large"
	UploadKindInternal        UploadErrorKind = "internal"

	defaultMaxClothingImageSize int64 = 10 * 1024 * 1024
)

type UploadError struct {
	Kind    UploadErrorKind
	Message string
	Err     error
}

func (err *UploadError) Error() string {
	if err.Err == nil {
		return err.Message
	}
	return err.Message + ": " + err.Err.Error()
}

func (err *UploadError) Unwrap() error {
	return err.Err
}

type UploadService struct {
	uploads              *manager.UploadManager
	maxClothingImageSize int64
}

func NewUploadService(uploads *manager.UploadManager) *UploadService {
	return &UploadService{
		uploads:              uploads,
		maxClothingImageSize: defaultMaxClothingImageSize,
	}
}

func (service *UploadService) UploadClothingImage(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (*dto.UploadFileResponse, error) {
	if service == nil || service.uploads == nil {
		return nil, newUploadError(UploadKindInternal, "上传服务未初始化", nil)
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, newUploadError(UploadKindUnauthorized, "未登录", nil)
	}

	if fileHeader == nil {
		return nil, newUploadError(UploadKindInvalidInput, "请选择要上传的衣物图片", nil)
	}

	if fileHeader.Size <= 0 {
		return nil, newUploadError(UploadKindInvalidInput, "上传文件不能为空", nil)
	}

	if fileHeader.Size > service.maxClothingImageSize {
		return nil, newUploadError(UploadKindTooLarge, "衣物图片不能超过10MB", nil)
	}

	uploaded, err := service.uploads.SaveClothingImage(ctx, manager.SaveUploadParams{
		UserID:     userID,
		FileHeader: fileHeader,
	})
	if err != nil {
		return nil, normalizeUploadManagerError(err)
	}

	response := toUploadFileResponse(uploaded)
	return &response, nil
}

func UploadKindOf(err error) UploadErrorKind {
	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr.Kind
	}
	return UploadKindInternal
}

func UploadMessageOf(err error) string {
	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr.Message
	}
	return "服务器内部错误"
}

func normalizeUploadManagerError(err error) error {
	if errors.Is(err, manager.ErrUploadFileEmpty) {
		return newUploadError(UploadKindInvalidInput, "上传文件不能为空", err)
	}

	if errors.Is(err, manager.ErrUnsupportedUploadType) {
		return newUploadError(UploadKindUnsupportedType, "仅支持 JPG、PNG、WEBP 格式的图片", err)
	}

	if errors.Is(err, manager.ErrInvalidUploadPath) {
		return newUploadError(UploadKindInvalidInput, "上传路径不合法", err)
	}

	if errors.Is(err, context.Canceled) {
		return newUploadError(UploadKindInternal, "上传已取消", nil)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return newUploadError(UploadKindInternal, "上传超时", nil)
	}

	return newUploadError(UploadKindInternal, "保存衣物图片失败", err)
}

func toUploadFileResponse(file *manager.UploadedFile) dto.UploadFileResponse {
	if file == nil {
		return dto.UploadFileResponse{}
	}

	return dto.UploadFileResponse{
		OriginalName: file.OriginalName,
		FileName:     file.FileName,
		ContentType:  file.ContentType,
		Size:         file.Size,
		RelativePath: file.RelativePath,
		ImageURL:     file.PublicURL,
		PublicURL:    file.PublicURL,
	}
}

func newUploadError(kind UploadErrorKind, message string, err error) *UploadError {
	return &UploadError{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}
