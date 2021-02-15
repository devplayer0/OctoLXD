package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Server) streamsIndex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stream, err := s.ss.GenerateStream(r.Context(), vars["owner"], vars["repo"])
	if err != nil {
		JSONErrResponse(w, err, 0)
		return
	}

	JSONResponse(w, stream, http.StatusOK)
}

func (s *Server) streamsImages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	images, err := s.ss.GenerateImages(r.Context(), vars["owner"], vars["repo"])
	if err != nil {
		JSONErrResponse(w, err, 0)
		return
	}

	JSONResponse(w, images, http.StatusOK)
}
