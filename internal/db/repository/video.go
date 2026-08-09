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
	queries     *sqlc.Queries
	transaction *Transaction
}

func NewVideoRepository(q *sqlc.Queries, t *Transaction) *VideoRepository {
	return &VideoRepository{queries: q, transaction: t}
}

func (vr *VideoRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	fmt.Printf("Repo::CreateVideo() ID: %s, Title: %s, OriginalFilename: %s, StoredFilename: %s\n", v.ID, v.Title, v.OriginalFilename,
		v.StoredFilename)

	vid_id, err := uuid.Parse(v.ID)
	if err != nil {
		return fmt.Errorf("failed to parse UUID: %w", err)
	}

	err = vr.transaction.ExecTx(ctx, func(q *sqlc.Queries) error {
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

		_, err := q.CreateVideo(ctx, params)
		if err != nil {
			fmt.Printf("Repository: q.CreateVideo returned error: %v\n", err)
			return fmt.Errorf("failed to create video: %w", err)
		}

		fmt.Printf("Repository: q.CreateVideo succeeded, no error.\n")
		return nil
	})

	return err
}

func (vr *VideoRepository) UpdateVideoState(ctx context.Context, id string, state domain.VideoState) error {
	fmt.Printf("Repo::UpdateVideoState() ID: %s, state: %s\n", id, state)

	vid_id, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("failed to parse UUID: %w", err)
	}
	err = vr.transaction.ExecTx(ctx, func(q *sqlc.Queries) error {
		params := sqlc.UpdateVideoStateParams{
			ID:    vid_id,
			State: string(state),
		}

		_, err := q.UpdateVideoState(ctx, params)
		if err != nil {
			fmt.Printf("Repository: q.UpdateVideoState returned error: %v\n", err)
			return fmt.Errorf("failed to update video state: %w", err)
		}

		fmt.Printf("Repository: q.UpdateVideoState succeeded, no error.\n")
		return nil
	})

	return err
}
