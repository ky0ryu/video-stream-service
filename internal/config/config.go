// Package config
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env          string // "dev" | "prod"
	LocalStorage string
	Port         string
	SizeLimitMB  int64
	DatabaseURL  string
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

	return &Config{
		Env:          getEnv("APP_DEV", "dev"),
		LocalStorage: getEnv("LOCAL_STORAGE_DIR", "./tmp/"),
		Port:         getEnv("PORT", "3000"),
		SizeLimitMB:  sizeLimitMB,
		DatabaseURL:  getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
