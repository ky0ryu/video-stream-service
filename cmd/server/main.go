package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/ky0ryu/video-upload-service/internal/api"
	"github.com/ky0ryu/video-upload-service/internal/config"
	"github.com/ky0ryu/video-upload-service/internal/db"
	"github.com/ky0ryu/video-upload-service/internal/db/repository"
	"github.com/ky0ryu/video-upload-service/internal/handler"
	"github.com/ky0ryu/video-upload-service/internal/service"
	sqlc "github.com/ky0ryu/video-upload-service/internal/sqlc/model"
	"github.com/ky0ryu/video-upload-service/internal/storage"
	"github.com/ky0ryu/video-upload-service/internal/validator"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

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

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
	defer func() {
		log.Println("Closing Asynq redis client...")
		asynqClient.Close()
	}()

	// server initialization
	store := storage.NewLocalStorage(cfg.LocalStorage)
	vtr := validator.NewVideoValidator(cfg.SizeLimitMB)
	service := service.NewVideoService(store, repo, vtr, asynqClient, cfg.SizeLimitMB)
	handler := handler.NewVideoHandler(service)
	srvr := api.NewServer(handler, cfg.Port)

	// the server is run asynchronously to allow graceful error handling
	go func() {
		if err := srvr.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed run: %v", err)
		}
	}()

	// setup a channel that'll wait for an OS interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-quit // blocking call
	// once a signal is received, function will proceed to shutdown the server

	log.Println("Shutting down server in 15 seconds (or less)...")
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer closeCancel()

	// signals the server to wrap up transactions before the 15s timeout is reached
	if err := srvr.Shutdown(closeCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server closed.")
}
