package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the octolxd server
type Server struct {
	config Config

	http *http.Server
}

// NewServer creates a new iamd server
func NewServer(config Config) *Server {
	router := mux.NewRouter()
	s := &Server{
		config: config,

		http: &http.Server{
			Addr:    config.HTTP.ListenAddress,
			Handler: router,
		},
	}

	router.HandleFunc("/health", s.healthCheck)

	//router.NotFoundHandler = http.HandlerFunc(s.apiNotFound)
	//router.MethodNotAllowedHandler = http.HandlerFunc(s.apiMethodNotAllowed)

	return s
}

// Start starts the octolxd server
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Stop shuts down the octolxd server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return s.http.Shutdown(ctx)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
