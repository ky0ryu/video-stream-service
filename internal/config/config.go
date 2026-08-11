// Package config
package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env              string // "dev" | "prod"
	LocalStorage     string
	Port             string
	SizeLimitMB      int64
	DatabaseURL      string
	RedisURL         string
	AsynqConcurrency int
}

func Load() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Print("Unable to load .env file")
	}

	sizeLimitMB, err := strconv.ParseInt(getEnv("SIZE_LIMIT_MB", "512"), 10, 64)
	if err != nil {
		log.Print("Failed to convert SIZE_LIMIT_MB, setting the value to default")
		sizeLimitMB = 512
	}

	// ensure that the absolute storage path is used
	storageDir := getEnv("STORAGE_DIR", "./tmp/")
	absPath, err := filepath.Abs(storageDir)
	if err == nil {
		storageDir = absPath
	}

	asynqConcurrency, err := strconv.Atoi(getEnv("ASYNQ_CONCURRENCY", "2"))
	if err != nil {
		log.Print("Failed to convert ASYNQ_CONCURRENCY, setting the value to default")
		asynqConcurrency = 2
	}

	return &Config{
		Env:              getEnv("APP_DEV", "dev"),
		LocalStorage:     storageDir,
		Port:             getEnv("PORT", "3000"),
		SizeLimitMB:      sizeLimitMB,
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		RedisURL:         getEnv("REDIS_URL", "redis:6379"),
		AsynqConcurrency: asynqConcurrency,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
