// Package repository
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ky0ryu/video-upload-service/internal/db/model"
	"github.com/ky0ryu/video-upload-service/internal/domain"
)

type VideoRepository struct {
	queries     *db.Queries
	transaction *Transaction
}

func NewVideoRepository(q *db.Queries, t *Transaction) *VideoRepository {
	return &VideoRepository{queries: q, transaction: t}
}

func (vr *VideoRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	fmt.Printf("Repository: CreateVideo called for Video ID: %s, Title: %s, OriginalFilename: %s, StoredFilename: %s\n", v.ID, v.Title, v.OriginalFilename,
		v.StoredFilename)
	vid_id, err := uuid.Parse(v.ID)
	if err != nil {
		return fmt.Errorf("failed to parse UUID: %w", err)
	}

	err = vr.transaction.ExecTx(ctx, func(q *db.Queries) error {
		time_now := time.Now()
		params := db.CreateVideoParams{
			ID:               vid_id,
			Title:            v.Title,
			OriginalFilename: v.OriginalFilename,
			StoredFilename:   v.StoredFilename,
			// Description:      dbDescription,
			CreatedAt: pgtype.Timestamptz{
				Time:  time_now,
				Valid: true,
			},
			UpdatedAt: pgtype.Timestamptz{
				Time:  time_now,
				Valid: true,
			},
		}
		fmt.Printf("Repository: Executing CreateVideo with params: %+v\n", params) // Print the params

		_, err := q.CreateVideo(ctx, params)
		if err != nil {
			fmt.Printf("Repository: q.CreateVideo returned error: %v\n", err) // Print the error
			return fmt.Errorf("failed to create video: %w", err)
		}

		fmt.Printf("Repository: q.CreateVideo succeeded, no error.\n")
		return nil
	})

	return err
}

func (vr *VideoRepository) UpdateVideoState(ctx context.Context, id string, state string, url string) error {
	return nil
}

func (vr *VideoRepository) GetVideo(ctx context.Context, id string) (*domain.Video, error) {
	return nil, nil
}
