// Package repository
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	"github.com/ky0ryu/video-upload-service/internal/sqlc/model"
)

type VideoRepository struct {
	queries *sqlc.Queries
}

func NewVideoRepository(q *sqlc.Queries) *VideoRepository {
	return &VideoRepository{queries: q}
}

func (vr *VideoRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	fmt.Printf("Repo::CreateVideo() ID: %s, Title: %s, OriginalFilename: %s, StoredFilename: %s\n", v.ID, v.Title, v.OriginalFilename,
		v.StoredFilename)

	vid_id, err := uuid.Parse(v.ID)
	if err != nil {
		return fmt.Errorf("failed to parse UUID: %w", err)
	}

	time_now := time.Now()
	params := sqlc.CreateVideoParams{
		ID:               vid_id,
		Title:            v.Title,
		OriginalFilename: v.OriginalFilename,
		StoredFilename:   v.StoredFilename,
		Description: pgtype.Text{
			String: v.Description, Valid: true,
		},
		State: string(domain.VideoPending),
		CreatedAt: pgtype.Timestamptz{
			Time:  time_now,
			Valid: true,
		},
		UpdatedAt: pgtype.Timestamptz{
			Time:  time_now,
			Valid: true,
		},
	}
	fmt.Printf("Repository: Executing CreateVideo with params: %+v\n", params) // prints with field names

	_, crtErr := vr.queries.CreateVideo(ctx, params)
	if crtErr != nil {
		fmt.Printf("Repository: q.CreateVideo returned error: %v\n", crtErr)
		return fmt.Errorf("failed to create video: %w", crtErr)
	}

	fmt.Printf("Repository: q.CreateVideo succeeded.\n")
	return nil
}

func (vr *VideoRepository) UpdateVideoState(ctx context.Context, id string, state domain.VideoState) error {
	fmt.Printf("Repo::UpdateVideoState() ID: %s, state: %s\n", id, state)

	vid_id, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("failed to parse UUID: %w", err)
	}
	params := sqlc.UpdateVideoStateParams{
		ID:    vid_id,
		State: string(state),
	}

	_, updErr := vr.queries.UpdateVideoState(ctx, params)
	if updErr != nil {
		fmt.Printf("Repository: q.UpdateVideoState returned error: %v\n", updErr)
		return fmt.Errorf("failed to update video state: %w", updErr)
	}

	fmt.Printf("Repository: q.UpdateVideoState succeeded.\n")
	return nil
}
