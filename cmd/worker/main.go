package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/ky0ryu/video-upload-service/internal/config"
	"github.com/ky0ryu/video-upload-service/internal/db"
	"github.com/ky0ryu/video-upload-service/internal/db/repository"
	sqlc "github.com/ky0ryu/video-upload-service/internal/sqlc/model"
	"github.com/ky0ryu/video-upload-service/internal/task"
	"github.com/ky0ryu/video-upload-service/internal/transcoder"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	fmt.Printf("Storage path: %v\n", cfg.LocalStorage)

	// DB initialization
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer func() {
		log.Println("Closing DB connection pool...")
		pool.Close()
	}()

	queries := sqlc.New(pool)
	repo := repository.NewVideoRepository(queries)
	tr := &transcoder.VideoTranscoder{Repo: repo}

	var redisOpt asynq.RedisConnOpt
	if strings.HasPrefix(cfg.RedisURL, "redis://") || strings.HasPrefix(cfg.RedisURL, "rediss://") {
		var err error
		redisOpt, err = asynq.ParseRedisURI(cfg.RedisURL)
		if err != nil {
			log.Fatalf("Unable to parse redis URL: %v\n", err)
		}
	} else {
		redisOpt = asynq.RedisClientOpt{Addr: cfg.RedisURL}
	}

	asSvc := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.AsynqConcurrency,
			Queues:      map[string]int{"transcode": 1},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeTranscodeVideoType, tr.TranscodeVideo)

	if err := asSvc.Run(mux); err != nil {
		log.Printf("worker exited: %v", err)
		// trigger the deferred funcs
		return
	}

	fmt.Println("Worker successfully exit.")
}
