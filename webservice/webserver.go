package webservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OmniCar/autobot/vehicle"
)

const (
	errJSONEncoding = iota * 100
	errLogRetrieval
)

type status struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

type storeStatus struct {
	HistorySize  int       `json:"historySize"`
	LastStatusAt time.Time `json:"lastStatusAt"`
	LastStatus   string    `json:"lastStatusMessage"`
}

// WebServer represents the REST-API part of autobot.
type WebServer struct {
	startTime time.Time
	store     *vehicle.Store
}

// APIError is the error returned to clients whenever an internal error has happened.
type APIError struct {
	HTTPCode int    `json:"-"`
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message"`
}

// New initialises a new webserver. You need to start it by calling Serve().
func New(store *vehicle.Store) *WebServer {
	return &WebServer{time.Now(), store}
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

// returnVehicleStoreStatus fetches and returns the current status of the vehicle store.
func (srv *WebServer) returnVehicleStoreStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	c, err := srv.store.CountLog()
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errLogRetrieval, err.Error()})
		return
	}
	entry, err := srv.store.LastLog()
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

// Serve starts the web server.
// Currently, there is no config for which port to listen on.
func (srv *WebServer) Serve(port uint) error {
	http.HandleFunc("/", srv.returnStatus)                                // GET.
	http.HandleFunc("/vehiclestore/status", srv.returnVehicleStoreStatus) // GET.
	srv.startTime = time.Now()
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
