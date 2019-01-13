package webservice

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mkock/autobot/vehicle"
)

type storeStatus struct {
	HistorySize  int       `json:"historySize"`
	LastStatusAt time.Time `json:"lastStatusAt"`
	LastStatus   string    `json:"lastStatusMessage"`
}

// handleStoreStatus fetches and returns the current status of the vehicle store.
func (srv *WebServer) handleStoreStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var entry vehicle.LogEntry
	c, err := srv.store.CountLog()
	if err == nil {
		entry, err = srv.store.LastLog()
	}
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errLogRetrieval, err.Error()})
		return
	}
	s := storeStatus{c, entry.LoggedAt, entry.Message}
	bytes, err := json.Marshal(s)
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errJSONEncoding, err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
