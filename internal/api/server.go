// Package api
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ky0ryu/video-upload-service/internal/handler"
)

type APIHandler interface {
	Upload(c *gin.Context)
}

type Server struct {
	engine   *gin.Engine
	port     string
	httpSrvr *http.Server
}

func sizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func NewServer(hdlr APIHandler, port string, sizeLimitMB int64) *Server {
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 20 // limit memory size to 8MB
	engine.Use(sizeLimit(sizeLimitMB << 20))
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(handler.ErrorHandler())

	engine.POST("/upload", hdlr.Upload)

	return &Server{
		engine: engine,
		port:   port,
	}
}

func (srvr *Server) Run() error {
	srvr.httpSrvr = &http.Server{
		Addr:         ":" + srvr.port,
		Handler:      srvr.engine,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	return srvr.httpSrvr.ListenAndServe()
}

func (srvr *Server) Shutdown(ctx context.Context) error {
	if srvr.httpSrvr != nil {
		return srvr.httpSrvr.Shutdown(ctx)
	}
	return nil
}
