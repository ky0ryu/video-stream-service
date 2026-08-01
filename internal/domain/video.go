// Package domain
package domain

import (
	"context"
	"io"
)

type Video struct {
	ID               string
	Title            string
	Description      string
	OriginalFilename string
	StoredFilename   string

	// State       string // uploading, processing, ready, failed
}

type VideoFile struct {
	Video
	File io.Reader
	Size int64
}

type VideoRepository interface {
	CreateVideo(ctx context.Context, v *Video) error
	// UpdateVideoState(ctx context.Context, id string, state string, url string) error
	GetVideo(ctx context.Context, id string) (*Video, error)
}
