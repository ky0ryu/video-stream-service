package main

import (
	"context"
	"log"

	"github.com/ky0ryu/video-upload-service/internal/api"
	"github.com/ky0ryu/video-upload-service/internal/config"
	"github.com/ky0ryu/video-upload-service/internal/db"
	"github.com/ky0ryu/video-upload-service/internal/db/repository"
	"github.com/ky0ryu/video-upload-service/internal/handler"
	"github.com/ky0ryu/video-upload-service/internal/service"
	sqlc "github.com/ky0ryu/video-upload-service/internal/sqlc/model"
	"github.com/ky0ryu/video-upload-service/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	tx := repository.NewTransaction(pool)
	repo := repository.NewVideoRepository(queries, tx)

	store := storage.NewLocalStorage(cfg.LocalStorage)
	service := service.NewVideoService(store, repo, cfg.SizeLimitMB)
	handler := handler.NewVideoHandler(service)
	srv := api.NewServer(handler, cfg.Port)

	// TODO: hook-up video service to video handler
	// vidSvc := service.NewVideoService(videoRepo)
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
