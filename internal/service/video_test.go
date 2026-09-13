package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ky0ryu/video-upload-service/internal/domain"
)

// Storage mock
type mockStorage struct {
	saveErr   error
	deleteErr error
	deleteID  string
}

func (m *mockStorage) Save(ctx context.Context, id string, filename string, rdr io.Reader, size int64) error {
	return m.saveErr
}

func (m *mockStorage) Delete(ctx context.Context, folderName string, fileName string) error {
	m.deleteID = folderName
	return m.deleteErr
}

func (m *mockStorage) GetFullFilePath(folderName string, fileName string) string {
	return "/tmp/" + folderName + "/" + fileName
}

// Repository mock
type mockRepository struct {
	createErr error
	video     *domain.Video
}

func (m *mockRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	m.video = v
	return m.createErr
}

func (m *mockRepository) UpdateVideoState(ctx context.Context, id string, state domain.VideoState) error {
	if m.video != nil && m.video.ID == id {
		m.video.State = state
	}
	return nil
}

// Validator mock
type mockValidator struct {
	validateErr error
}

func (m *mockValidator) Validate(vf *domain.VideoFile) error {
	return m.validateErr
}

func TestVideoService_UploadVideo_RollbackCompensation(t *testing.T) {
	t.Run("DB Create fails triggers file deletion", func(t *testing.T) {
		store := &mockStorage{}
		repo := &mockRepository{createErr: errors.New("database connection lost")}
		val := &mockValidator{}

		// Instantiate service under test (Asynq client left as nil for mock run)
		svc := NewVideoService(store, repo, val, nil, 10)

		vf := domain.VideoFile{
			Video: domain.Video{
				Title:            "My Test Video",
				OriginalFilename: "video.mp4",
			},
			File: nil,
			Size: 100,
		}

		err := svc.UploadVideo(context.Background(), vf)

		if err == nil {
			t.Fatal("expected UploadVideo to fail, but it return nil")
		}

		// Verify deletion compensation was executed to prevent orphaned files
		if store.deleteID == "" {
			t.Error("expected mockStorage.Delete to be triggered on DB failure, but it was not")
		}
	})
}
