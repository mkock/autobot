package webservice

import (
	"encoding/json"
	"net/http"
	"time"
)

type status struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// handleStatus returns a small JSON struct with the various information such as service uptime and status.
func (srv *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
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
