package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/devplayer0/octolxd/pkg/simplestreams"
	"github.com/google/go-github/v37/github"
	"github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
)

// ErrToStatus gets the HTTP statuscode for an error
func ErrToStatus(err error) int {
	var ghError *github.ErrorResponse
	switch {
	case errors.Is(err, simplestreams.ErrInvalidPath):
		return http.StatusNotFound
	case errors.As(err, &ghError):
		return ghError.Response.StatusCode
	default:
		return http.StatusInternalServerError
	}
}

// JSONResponse Sends a JSON payload in response to a HTTP request
func JSONResponse(w http.ResponseWriter, v interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		log.WithField("err", err).Error("Failed to serialize JSON payload")

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Failed to serialize JSON payload")
	}
}

type jsonError struct {
	Message string `json:"message"`
}

// JSONErrResponse Sends an `error` as a JSON object with a `message` property
func JSONErrResponse(w http.ResponseWriter, err error, statusCode int) {
	log.WithError(err).Error("Error while processing request")

	w.Header().Set("Content-Type", "application/problem+json")
	if statusCode == 0 {
		statusCode = ErrToStatus(err)
	}
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	enc.Encode(jsonError{err.Error()})
}

func writeAccessLog(w io.Writer, params handlers.LogFormatterParams) {
	log.WithFields(log.Fields{
		"agent":   params.Request.UserAgent(),
		"status":  params.StatusCode,
		"resSize": params.Size,
	}).Debugf("%v %v", params.Request.Method, params.URL.RequestURI())
}
