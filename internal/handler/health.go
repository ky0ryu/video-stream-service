package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, rdc *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rdc}
}

// Live checks whether the server is running or not (shallow check)
func (h *HealthHandler) Live(c *gin.Context) {
	c.Status(http.StatusOK)
}

// Ready checks for the dependency reachability
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// check db availability
	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "down",
			"database": "unreachable",
		})
		return
	}

	// check redis availability
	if err := h.redis.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "down",
			"redis":  "unreachable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
