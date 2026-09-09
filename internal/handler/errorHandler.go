package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ky0ryu/video-upload-service/internal/response"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // perform other handlers first

		if len(c.Errors) == 0 {
			return
		}

		// prevent overwriting headers/body
		if c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err

		var apiErr *response.APIError
		if errors.As(err, &apiErr) {
			if apiErr.Err != nil {
				log.Printf("request failed: %v", apiErr.Err)
			}
			log.Printf("Returning error status: %d, message: %s", apiErr.StatusCode, apiErr.Message)
			// abort or stop the context and then return the status error code
			c.AbortWithStatusJSON(apiErr.StatusCode, gin.H{"error": apiErr.Message})
			return
		}
		// fallback: abort or stop the context and then return Status code 500
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
	}
}
