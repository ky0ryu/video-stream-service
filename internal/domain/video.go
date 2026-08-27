// Package domain
package domain

import (
	"context"
	"io"
)

type VideoState string

const (
	VideoPending     VideoState = "pending"     // video upload started (default state)
	VideoUploading   VideoState = "uploading"   // video upload is ongoing
	VideoTranscoding VideoState = "transcoding" // video transcode is ongoing
	VideoFailed      VideoState = "failed"      // upload or transcode had failed
	VideoDeleted     VideoState = "deleted"     // file has been deleted
	VideoReady       VideoState = "ready"       // video is ready for playback
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

type VideoValidator interface {
	Validate(vf *VideoFile) error
}
