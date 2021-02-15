package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/devplayer0/octolxd/pkg/simplestreams"
)

// Server represents the octolxd server
type Server struct {
	config Config

	ss   *simplestreams.SimpleStreams
	http *http.Server
}

// NewServer creates a new iamd server
func NewServer(config Config) *Server {
	router := mux.NewRouter()
	s := &Server{
		config: config,

		ss: simplestreams.NewSimpleStreams(),
		http: &http.Server{
			Addr:    config.HTTP.ListenAddress,
			Handler: handlers.CustomLoggingHandler(nil, router, writeAccessLog),
		},
	}

	router.HandleFunc("/health", s.healthCheck)
	repoRouter := router.PathPrefix("/{owner}/{repo}").Subrouter()

	streamsRouter := repoRouter.PathPrefix("/streams/v1").Subrouter()
	streamsRouter.HandleFunc("/index.json", s.streamsIndex).Methods(http.MethodGet)
	streamsRouter.HandleFunc("/images.json", s.streamsImages).Methods(http.MethodGet)

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
