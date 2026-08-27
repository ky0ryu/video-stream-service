// Package service
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	"github.com/ky0ryu/video-upload-service/internal/storage"
	"github.com/ky0ryu/video-upload-service/internal/task"
)

type VideoService struct {
	store       storage.Storage
	repo        domain.VideoRepository
	validator   domain.VideoValidator
	asyncClient *asynq.Client
	sizeLimitMB int64
}

var allowedVidExt = map[string]bool{
	".mp4": true,
	".mov": true,
	".mkv": true,
	".avi": true,
}

func NewVideoService(s storage.Storage, r domain.VideoRepository, vv domain.VideoValidator, ac *asynq.Client, sl int64) *VideoService {
	return &VideoService{store: s, repo: r, validator: vv, asyncClient: ac, sizeLimitMB: sl}
}

func (svc *VideoService) UploadVideo(ctx context.Context, vf domain.VideoFile) error {

	if err := svc.validator.Validate(&vf); err != nil {
		return fmt.Errorf("video validation failed: %w", err)
	}

	// generate unique video ID
	vf.ID = uuid.New().String()
	ext := filepath.Ext(vf.OriginalFilename)
	vf.StoredFilename = vf.ID + ext

	// save the file to the specified storage
	if err := svc.store.Save(ctx, vf.ID, vf.StoredFilename, vf.File, vf.Size); err != nil {
		return fmt.Errorf("failed to save video file: %w", err)
	}

	// save the video data to db
	if err := svc.repo.CreateVideo(ctx, &vf.Video); err != nil {
		// Delete the file incase the DB transaction fails
		if delErr := svc.store.Delete(ctx, vf.ID, vf.StoredFilename); delErr != nil {
			fmt.Printf("failed to delete uploaded file: %s: %v", vf.StoredFilename, delErr)
		}
		return fmt.Errorf("failed to create video data in DB: %w", err)
	}

	if err := svc.createTranscodeTask(vf.Video); err != nil {
		// Delete the file when transcode enqueue fails
		if delErr := svc.store.Delete(ctx, vf.ID, vf.StoredFilename); delErr != nil {
			fmt.Printf("failed to delete uploaded file: %s: %v", vf.StoredFilename, delErr)
		}

		if updErr := svc.repo.UpdateVideoState(ctx, vf.ID, domain.VideoDeleted); updErr != nil {
			fmt.Printf("failed to update state: %v", updErr)
		}
		return fmt.Errorf("failed to send transcode video task: %w", err)
	}

	return nil
}

func (svc *VideoService) createTranscodeTask(video domain.Video) error {
	filePath := svc.store.GetFullFilePath(video.ID, video.StoredFilename)
	payload, _ := json.Marshal(task.TranscodeVideoPayload{
		VideoID: video.ID,
		SaveDir: filePath,
	})
	t := asynq.NewTask(task.TypeTranscodeVideoType, payload)

	_, err := svc.asyncClient.Enqueue(t, asynq.MaxRetry(3), asynq.Queue("transcode"))
	if err != nil {
		return fmt.Errorf("failed to Enqueue: %w", err)
	}

	return nil
}
