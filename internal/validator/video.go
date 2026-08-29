package validator

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/ky0ryu/video-upload-service/internal/domain"
)

var allowedVidExt = map[string]bool{
	".mp4": true,
	".mov": true,
	".mkv": true,
	".avi": true,
}

var allowedVideos = map[string]bool{
	"video/mp4": true,
	"video/mov": true,
	"video/mkv": true,
	"video/avi": true,
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

	mtype, err := mimetype.DetectReader(vf.File)
	if err != nil {
		return fmt.Errorf("invalid video file: %w", err)
	}

	// make sure that vf.File can be asserted to ReadSeeker
	if seeker, ok := vf.File.(io.ReadSeeker); ok {
		// reset the seeker at the beginning of the stream
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to rollback file reader: %w", err)
		}
	}

	if !allowedVideos[mtype.String()] {
		return errors.New("unsupported video format")
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
