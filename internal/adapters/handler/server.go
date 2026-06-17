package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func NewServer(
	router chi.Router,
	serverAddr string,
	readHeaderTimeout,
	readTimeout,
	writeTimeout,
	idleTimeout int,
) *http.Server {
	return &http.Server{
		Addr:              serverAddr,
		Handler:           router,
		ReadHeaderTimeout: time.Duration(readHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(readTimeout) * time.Second,
		WriteTimeout:      time.Duration(writeTimeout) * time.Second,
		IdleTimeout:       time.Duration(idleTimeout) * time.Second,
	}
}
