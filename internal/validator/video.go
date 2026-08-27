package validator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ky0ryu/video-upload-service/internal/domain"
)

var allowedVidExt = map[string]bool{
	".mp4": true,
	".mov": true,
	".mkv": true,
	".avi": true,
}

type VideoValidator struct {
	sizeLimitMB int64
}

func NewVideoValidator(sizeLimitMB int64) *VideoValidator {
	return &VideoValidator{
		sizeLimitMB: sizeLimitMB,
	}
}

func (vv *VideoValidator) Validate(vf *domain.VideoFile) error {
	if err := checkFileSize(vf.Size, vv.sizeLimitMB); err != nil {
		return err
	}

	if err := checkFileFormat(vf.OriginalFilename); err != nil {
		return err
	}
	return nil
}

func checkFileSize(size int64, sizeLimitMB int64) error {
	maxSizeB := sizeLimitMB * 1024 * 1024

	if size > maxSizeB {
		return fmt.Errorf("file exceeds %d byte limit", maxSizeB)
	}
	return nil
}

func checkFileFormat(filaName string) error {
	ext := strings.ToLower(filepath.Ext(filaName))
	if !allowedVidExt[ext] {
		return fmt.Errorf("unsupported format: %s", ext)
	}

	return nil
}
