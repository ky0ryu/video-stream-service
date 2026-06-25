// Package handler
package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ky0ryu/video-upload-service/internal/storage"
)

var allowedVidExt = map[string]bool{
	".mp4": true,
	".mov": true,
	".mkv": true,
	".avi": true,
}

type VideoHandler struct {
	store       storage.Storage
	sizeLimitMB int64
}

func NewVideoHandler(store storage.Storage, sizeLimitMB int64) *VideoHandler {
	return &VideoHandler{store: store, sizeLimitMB: sizeLimitMB}
}

func (v *VideoHandler) Upload(c *gin.Context) {
	// limit the total request body size to prevent disk space overloading
	maxBytes := (v.sizeLimitMB + 1) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	// multipart file parse
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file required",
		})
		return
	}

	// validate
	if err := validate(fileHeader, v.sizeLimitMB); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	// open
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to open file",
		})
		return
	}
	defer file.Close()

	// save
	jobID := uuid.NewString()
	ext := filepath.Ext(fileHeader.Filename)
	filename := jobID + ext

	if err := v.store.Save(c.Request.Context(), filename, file, fileHeader.Size); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "storage failure",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func validate(fileHeader *multipart.FileHeader, sizeLimitMB int64) error {
	maxSizeB := sizeLimitMB * 1024 * 1024

	if fileHeader.Size > maxSizeB {
		return fmt.Errorf("file exceeds %d byte limit", maxSizeB)
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedVidExt[ext] {
		return fmt.Errorf("unsupported format: %s", ext)
	}

	return nil
}
