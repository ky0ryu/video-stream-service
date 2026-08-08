// Package api
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIHandler interface {
	Upload(c *gin.Context)
}

type Server struct {
	engine   *gin.Engine
	port     string
	httpSrvr *http.Server
}

func NewServer(handler APIHandler, port string) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	engine.POST("/upload", handler.Upload)

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
		return srvr.Shutdown(ctx)
	}
	return nil
}
