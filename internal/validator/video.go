package validator

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/ky0ryu/video-upload-service/internal/domain"
)

type formatDef struct {
	ext   string
	mimes []string
}

var formats = []formatDef{
	{".mp4", []string{"video/mp4"}},
	{".mov", []string{"video/quicktime"}},
	{".mkv", []string{"video/matroska", "video/x-matroska"}},
	{".avi", []string{"video/x-msvideo", "video/avi"}},
}

var ErrTooLarge = errors.New("video exceeds file limit")

type VideoValidator struct {
	sizeLimitMB int64
}

func NewVideoValidator(sizeLimitMB int64) *VideoValidator {
	return &VideoValidator{
		sizeLimitMB: sizeLimitMB,
	}
}

func formatByExt(ext string) (formatDef, bool) {
	idx := slices.IndexFunc(formats, func(f formatDef) bool { return f.ext == ext })
	if idx < 0 {
		return formatDef{}, false
	}
	return formats[idx], true
}

func (vv *VideoValidator) Validate(vf *domain.VideoFile) error {
	ext := strings.ToLower(filepath.Ext(vf.OriginalFilename))
	f, ok := formatByExt(ext)
	if !ok {
		return fmt.Errorf("unsupported format: %s", filepath.Ext(vf.OriginalFilename))
	}

	mtype, err := mimetype.DetectReader(vf.File)
	if err != nil {
		return fmt.Errorf("invalid video file: %w", err)
	}

	// make sure that vf.File can be asserted to ReadSeeker
	seeker, ok := vf.File.(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("validator requires an io.ReadSeeker")
	}
	// reset the seeker at the beginning of the stream
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rollback file reader: %w", err)
	}

	// confirm that the read container matches the filename
	if !slices.ContainsFunc(f.mimes, mtype.Is) {
		return fmt.Errorf("unsupported video format: %s sent as %s", mtype, f.ext)
	}

	return nil
}

func (vv *VideoValidator) CheckFileSize(size int64) error {
	maxSizeB := vv.sizeLimitMB * 1024 * 1024

	if size > maxSizeB {
		return fmt.Errorf("%w: file size is %d bytes, limit is %d", ErrTooLarge, size, maxSizeB)
	}
	return nil
}
