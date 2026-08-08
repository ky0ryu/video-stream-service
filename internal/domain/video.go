// Package domain
package domain

import (
	"context"
	"io"
)

type VideoState string

const (
	VideoPending     VideoState = "pending"
	VideoUploading   VideoState = "uploading"
	VideoTranscoding VideoState = "transcoding"
	VideoReady       VideoState = "ready"
)

type Video struct {
	ID               string
	Title            string
	Description      string
	State            VideoState
	OriginalFilename string
	StoredFilename   string
}

type VideoFile struct {
	Video
	File io.Reader
	Size int64
}

type VideoRepository interface {
	CreateVideo(ctx context.Context, v *Video) error
	UpdateVideoState(ctx context.Context, id string, state VideoState) error
	// GetVideo(ctx context.Context, id string) (*Video, error)
}
