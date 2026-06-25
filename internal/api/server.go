// Package api
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIHandler interface {
	Upload(c *gin.Context)
}

type Server struct {
	engine *gin.Engine
	port   string
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

func (srv *Server) Run() error {
	httpSrv := &http.Server{
		Addr:         ":" + srv.port,
		Handler:      srv.engine,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	return httpSrv.ListenAndServe()
}
