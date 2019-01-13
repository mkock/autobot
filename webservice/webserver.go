package webservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/extlookup"
	"github.com/mkock/autobot/scheduler"
	"github.com/mkock/autobot/vehicle"
)

const (
	dateFmt = "2006-01-02"
	timeFmt = "2006-01-02T15:04:05"
)

const (
	errJSONEncoding = iota * 100
	errLogRetrieval
	errLookup
	errMarshalling
	errVehicleOp
)

type status struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// WebServer represents the REST-API part of autobot.
type WebServer struct {
	startTime  time.Time
	store      *vehicle.Store
	lookupMngr *extlookup.Manager
	cnf        config.Config
}

// APIError is the error returned to clients whenever an internal error has happened.
type APIError struct {
	HTTPCode int    `json:"-"`
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message"`
}

// APIVehicle is the API representation of Vehicle. It has a JSON representation.
// Some fields that are only for internal use, are left out, and others are converted into something more readable.
type APIVehicle struct {
	Hash         string `json:"hash"`
	Country      string `json:"country"`
	Type         string `json:"type"`
	RegNr        string `json:"regNr"`
	VIN          string `json:"vin"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Variant      string `json:"variant"`
	FuelType     string `json:"fuelType"`
	FirstRegDate string `json:"firstRegDate"`
	FromCache    bool   `json:"fromCache"`
}

// New initialises a new webserver. You need to start it by calling Serve().
func New(store *vehicle.Store, mngr *extlookup.Manager, cnf config.Config) *WebServer {
	return &WebServer{time.Now(), store, mngr, cnf}
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

// setupMux registers all the endpoints that the web server makes available.
func (srv *WebServer) setupMux() {
	http.HandleFunc("/", srv.handleStatus)                         // GET.
	http.HandleFunc("/vehiclestore/status", srv.handleStoreStatus) // GET.
	http.HandleFunc("/lookup", srv.handleLookup)                   // GET.
	http.HandleFunc("/vehicle", srv.handleVehicle)                 // PATCH.
}

// Serve starts the web server. It never returns unless interrupted.
func (srv *WebServer) Serve(port uint, sync bool) error {
	srv.setupMux()
	srv.startTime = time.Now()
	// Start a go routine with the web server.
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}()
	// Prepare a channel for service interruption using SIGINT/SIGTERM.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	if sync {
		// Start a go routine with the scheduler.
		sched := scheduler.New(srv.cnf, srv.store, os.Stdout)
		stop, err := sched.Start()
		if err != nil {
			return err // This will happen if the time expression from the config file couldn't be parsed.
		}
		defer func() {
			stop <- true
		}()
	}
	<-sigs // Function will halt here until interrupted.
	fmt.Println("\nInterrupted o_O")
	return nil
}
