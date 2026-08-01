package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ky0ryu/video-upload-service/internal/api"
	"github.com/ky0ryu/video-upload-service/internal/config"
	db "github.com/ky0ryu/video-upload-service/internal/db/model"
	"github.com/ky0ryu/video-upload-service/internal/handler"
	"github.com/ky0ryu/video-upload-service/internal/repository"
	"github.com/ky0ryu/video-upload-service/internal/service"
	"github.com/ky0ryu/video-upload-service/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err_db := pgxpool.New(ctx, "postgres://user:password@localhost:5432/video_db?sslmode=disable")
	if err_db != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err_db)
	}
	defer pool.Close()
	queries := db.New(pool)
	tx := repository.NewTransaction(pool)
	repo := repository.NewVideoRepository(queries, tx)

	store := storage.NewLocalStorage(cfg.LocalStorage)
	service := service.NewVideoService(store, repo, cfg.SizeLimitMB)
	handler := handler.NewVideoHandler(service)
	srv := api.NewServer(handler, cfg.Port)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("DB cannot be reached: %v\n", err)
	}

	// TODO: hook-up video service to video handler
	// vidSvc := service.NewVideoService(videoRepo)
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
