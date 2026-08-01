// Package service
package service

import (
	"context"
	"fmt"
	"strings"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	"github.com/ky0ryu/video-upload-service/internal/storage"
)

type VideoService struct {
	store       storage.Storage
	repo        domain.VideoRepository
	sizeLimitMB int64
}

var allowedVidExt = map[string]bool{
	".mp4": true,
	".mov": true,
	".mkv": true,
	".avi": true,
}

func NewVideoService(s storage.Storage, r domain.VideoRepository, sl int64) *VideoService {
	return &VideoService{store: s, repo: r, sizeLimitMB: sl}
}

func (svc *VideoService) UploadVideo(ctx context.Context, vf domain.VideoFile) error {

	// validate the video file
	if err := validate(vf.OriginalFilename, vf.Size, svc.sizeLimitMB); err != nil {
		return fmt.Errorf("video validation failed: %w", err)
	}

	// generate unique video ID
	vf.ID = uuid.New().String()
	ext := filepath.Ext(vf.OriginalFilename)
	vf.StoredFilename = vf.ID + ext

	// save the file to the specified storage
	if err := svc.store.Save(ctx, vf.StoredFilename, vf.File, vf.Size); err != nil {
		return fmt.Errorf("failed to save video file: %w", err)
	}

	// save the video data to db
	if err := svc.repo.CreateVideo(ctx, &vf.Video); err != nil {
		// Rollback DB: delete the file incase the DB transaction fails
		if delErr := svc.store.Delete(ctx, vf.StoredFilename); delErr != nil {
			fmt.Printf("failed to delete uploaded file: %s: %v", vf.StoredFilename, delErr)
		}
		return fmt.Errorf("failed to create video in DB: %w", err)
	}

	return nil
}

func validate(filename string, size int64, sizeLimitMB int64) error {
	maxSizeB := sizeLimitMB * 1024 * 1024

	if size > maxSizeB {
		return fmt.Errorf("file exceeds %d byte limit", maxSizeB)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedVidExt[ext] {
		return fmt.Errorf("unsupported format: %s", ext)
	}

	return nil
}

// func (s *VideoService) MarkTranscoded(ctx context.Context, id uuid.UUID) error {
//     return s.repo.UpdateStatus(ctx, id, domain.StatusReady)
// }
