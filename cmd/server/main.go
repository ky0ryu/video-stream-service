package main

import (
	"log"

	"github.com/ky0ryu/video-upload-service/internal/api"
	"github.com/ky0ryu/video-upload-service/internal/config"
	"github.com/ky0ryu/video-upload-service/internal/handler"
	"github.com/ky0ryu/video-upload-service/internal/storage"
)

func main() {
	cfg := config.Load()
	store := storage.NewLocalStorage(cfg.LocalStorage)
	handler := handler.NewVideoHandler(store, cfg.SizeLimitMB)
	srv := api.NewServer(handler, cfg.Port)

	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
