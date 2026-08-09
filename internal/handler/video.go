// Package handler
package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	"github.com/ky0ryu/video-upload-service/internal/service"
)

type VideoHandler struct {
	service *service.VideoService
}

func NewVideoHandler(s *service.VideoService) *VideoHandler {
	return &VideoHandler{service: s}
}

func (v *VideoHandler) Upload(ctx *gin.Context) {

	// multipart file parse
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "file required",
		})
		return
	}

	// open
	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to open file",
		})
		return
	}
	defer file.Close()

	// save
	title := ctx.PostForm("title")
	desc := ctx.PostForm("description")
	size := fileHeader.Size
	filename := fileHeader.Filename

	video_file := domain.VideoFile{
		Video: domain.Video{
			Title:            title,
			Description:      desc,
			OriginalFilename: filename,
		},
		File: file,
		Size: size,
	}

	if err := v.service.UploadVideo(ctx.Request.Context(), video_file); err != nil {
		// TODO: implement custom errors
		//     if errors.Is(err, service.ErrValidationFailed) { // Example custom error
		//         ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		//     } else if errors.Is(err, service.ErrStorageFailed) { // Example custom error
		//         ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store video"})
		//     } else {
		//         ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		//     }

		// error from UploadVideo() should not be return to the API caller
		log.Printf("UploadVideo failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong on the server",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
