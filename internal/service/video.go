// Package service
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	apierror "github.com/ky0ryu/video-upload-service/internal/response"
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

func NewVideoService(s storage.Storage, r domain.VideoRepository, vv domain.VideoValidator, ac *asynq.Client, sl int64) *VideoService {
	return &VideoService{store: s, repo: r, validator: vv, asyncClient: ac, sizeLimitMB: sl}
}

func (svc *VideoService) UploadVideo(ctx context.Context, vf domain.VideoFile) (string, string, error) {

	if err := svc.validator.Validate(&vf); err != nil {
		log.Printf("video validation failed: %v", err)
		return "", string(domain.VideoFailed), apierror.UnsupportedMediaType(err)
	}

	if err := svc.validator.CheckFileSize(vf.Size); err != nil {
		log.Printf("size went over the limit: %v", err)
		return "", string(domain.VideoFailed), apierror.PayloadTooLarge(err)
	}

	// generate unique video ID
	vf.ID = uuid.New().String()
	vf.State = domain.VideoPending
	ext := strings.ToLower(filepath.Ext(vf.OriginalFilename))
	vf.StoredFilename = vf.ID + ext

	// save the file to the specified storage
	if err := svc.store.Save(ctx, vf.ID, vf.StoredFilename, vf.File, vf.Size); err != nil {
		log.Printf("failed to save video file: %v", err)
		return "", "", apierror.InternalServerError(err)
	}

	// save the video data to db
	if err := svc.repo.CreateVideo(ctx, &vf.Video); err != nil {
		// ensure that file cleanup survives caller cancellation
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		// Delete the file incase the DB transaction fails
		if delErr := svc.store.Delete(cleanupCtx, vf.ID, vf.StoredFilename); delErr != nil {
			log.Printf("failed to delete uploaded file: %s: %v", vf.StoredFilename, delErr)
			return "", "", apierror.InternalServerError(delErr)
		}
		log.Printf("failed to create video data in DB: %v", err)
		return "", "", apierror.InternalServerError(err)
	}

	if err := svc.createTranscodeTask(vf.Video); err != nil {
		// ensure that file cleanup survives caller cancellation
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		// Delete the file when transcode enqueue fails
		if delErr := svc.store.Delete(cleanupCtx, vf.ID, vf.StoredFilename); delErr != nil {
			log.Printf("failed to delete uploaded file: %s: %v", vf.StoredFilename, delErr)
			return "", "", apierror.InternalServerError(delErr)
		}

		if updErr := svc.repo.UpdateVideoState(cleanupCtx, vf.ID, domain.VideoDeleted); updErr != nil {
			log.Printf("failed to update state: %v", updErr)
			return "", "", apierror.InternalServerError(updErr)
		}

		log.Printf("failed to send transcode video task: %v", err)
		return "", "", apierror.InternalServerError(err)
	}

	return vf.ID, string(vf.State), nil
}

func (svc *VideoService) createTranscodeTask(video domain.Video) error {
	filePath := svc.store.GetFullFilePath(video.ID, video.StoredFilename)
	payload, _ := json.Marshal(task.TranscodeVideoPayload{
		VideoID: video.ID,
		SaveDir: filePath,
	})
	t := asynq.NewTask(task.TypeTranscodeVideoType, payload)

	_, err := svc.asyncClient.Enqueue(t, asynq.Queue("transcode"), asynq.MaxRetry(3), asynq.Timeout(90*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to Enqueue: %w", err)
	}

	return nil
}
