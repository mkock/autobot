package webservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	errJSONEncoding = iota * 100
)

type status struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// WebServer represents the REST-API part of autobot.
type WebServer struct {
	startTime time.Time
}

// APIError is the error returned to clients whenever an internal error has happened.
type APIError struct {
	HTTPCode int    `json:"-"`
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message"`
}

// New initialises a new webserver. You need to start it by calling Serve().
func New() *WebServer {
	return &WebServer{}
}

// JSONError serves the given error as JSON.
func (srv *WebServer) JSONError(w http.ResponseWriter, handlerErr APIError) {
	data := struct {
		Err APIError `json:"error"`
	}{handlerErr}
	d, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(handlerErr.HTTPCode)
	fmt.Fprint(w, string(d))
}

// returnStatus returns a small JSON struct with the various information such as service uptime and status.
func (srv *WebServer) returnStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(srv.startTime).Truncate(time.Second)
	s := status{"running", uptime.String()}
	bytes, err := json.Marshal(s)
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errJSONEncoding, err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

// Serve starts the web server.
// Currently, there is no config for which port to listen on.
func (srv *WebServer) Serve(port uint) error {
	http.HandleFunc("/", srv.returnStatus)
	srv.startTime = time.Now()
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
