// Package handler
package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	apiResponse "github.com/ky0ryu/video-upload-service/internal/response"
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
		ctx.Error(apiResponse.BadRequest(err))
		return
	}

	// open
	file, err := fileHeader.Open()
	if err != nil {
		ctx.Error(apiResponse.InternalServerError(err))
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

		log.Printf("UploadVideo failed: %v", err)
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"status": "created",
	})
}
