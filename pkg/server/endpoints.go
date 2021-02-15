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

func (s *Server) imagesRedirect(w http.ResponseWriter, r *http.Request) {
	vs := mux.Vars(r)
	url, err := s.ss.GetPathURL(r.Context(), vs["owner"], vs["repo"], vs["name"], vs["arch"], vs["version"], vs["file"])
	if err != nil {
		JSONErrResponse(w, err, 0)
		return
	}

	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
