package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"volunteer-system/pkg/storage"

	"github.com/google/uuid"
)

var (
	ErrInvalidUploadType = errors.New("无效的上传类型")
	ErrFileTooLarge      = errors.New("文件大小超出限制")
	ErrInvalidExtension  = errors.New("不支持的文件格式")
)

type UploadType string

const (
	UploadTypeAvatar   UploadType = "avatar"
	UploadTypeActivity UploadType = "activity"
	UploadTypeOrg      UploadType = "org"
)

var uploadTypeDirs = map[UploadType]string{
	UploadTypeAvatar:   "avatar",
	UploadTypeActivity: "activity",
	UploadTypeOrg:      "org",
}

type UploadService struct {
	storage           storage.Storage
	maxFileSizeMB     int
	allowedExtensions []string
}

func NewUploadService(st storage.Storage, maxFileSizeMB int, allowedExts []string) *UploadService {
	return &UploadService{
		storage:           st,
		maxFileSizeMB:     maxFileSizeMB,
		allowedExtensions: allowedExts,
	}
}

func (s *UploadService) Upload(ctx context.Context, file io.Reader, filename string, fileSize int64, uploadType string) (string, error) {
	ut := UploadType(uploadType)
	dir, ok := uploadTypeDirs[ut]
	if !ok {
		return "", ErrInvalidUploadType
	}

	if s.maxFileSizeMB > 0 && fileSize > int64(s.maxFileSizeMB)*1024*1024 {
		return "", ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !s.isExtensionAllowed(ext) {
		return "", ErrInvalidExtension
	}

	savedPath := fmt.Sprintf("%s/%s%s", dir, uuid.New().String(), ext)
	if err := s.storage.Save(ctx, savedPath, file); err != nil {
		return "", err
	}
	return s.storage.GetURL(savedPath), nil
}

func (s *UploadService) isExtensionAllowed(ext string) bool {
	if len(s.allowedExtensions) == 0 {
		return true
	}
	for _, allowed := range s.allowedExtensions {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}
