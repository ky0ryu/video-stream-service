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
	// make sure that defers from main processes run
	os.Exit(run())
}

func run() int {
	exitCode := 0
	cfg := config.Load()
	ctx := context.Background()

	// DB initialization
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Unable to create connection pool: %v\n", err)
		exitCode = 1
		return exitCode
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

	if err := os.MkdirAll(cfg.LocalStorage, 0755); err != nil {
		log.Printf("failed to create dir: %v %v", cfg.LocalStorage, err)
		exitCode = 1
		return exitCode
	}

	// server initialization
	store := storage.NewLocalStorage(cfg.LocalStorage)
	vtr := validator.NewVideoValidator(cfg.SizeLimitMB)
	service := service.NewVideoService(store, repo, vtr, asynqClient, cfg.SizeLimitMB)
	handler := handler.NewVideoHandler(service)
	srvr := api.NewServer(handler, cfg.Port, cfg.SizeLimitMB)

	runErr := make(chan error, 1)
	go func() {
		if err := srvr.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-quit:
		log.Printf("received signal: %s", sig)
	case err := <-runErr:
		log.Printf("server failed run: %v", err)
		exitCode = 1
	}

	log.Println("Shutting down server in 15 seconds (or less)...")
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer closeCancel()

	// signals the server to wrap up transactions before the 15s timeout is reached
	if err := srvr.Shutdown(closeCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		exitCode = 1
	}

	log.Println("Server closed.")
	return exitCode
}
