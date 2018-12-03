package webservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

type storeStatus struct {
	HistorySize  int       `json:"historySize"`
	LastStatusAt time.Time `json:"lastStatusAt"`
	LastStatus   string    `json:"lastStatusMessage"`
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
	RegNr        string `json:"regNr"`
	VIN          string `json:"vin"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	FuelType     string `json:"fuelType"`
	FirstRegDate string `json:"firstRegDate"`
	FromCache    bool   `json:"fromCache"`
}

// vehicleToAPIType converts a vehicle.Vehicle into the local APIVehicle, which is used for the http request/response.
func vehicleToAPIType(veh vehicle.Vehicle, fromCache bool) APIVehicle {
	return APIVehicle{strconv.FormatUint(veh.MetaData.Hash, 10), vehicle.RegCountryToString(veh.MetaData.Country), veh.RegNr, veh.VIN, veh.Brand, veh.Model, veh.FuelType, veh.FirstRegDate.Format(dateFmt), fromCache}
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

// handleStoreStatus fetches and returns the current status of the vehicle store.
func (srv *WebServer) handleStoreStatus(w http.ResponseWriter, r *http.Request) {
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

// handleLookup allows vehicle lookups based on hash value, VIN or registration number. A country must always be
// provided.
func (srv *WebServer) handleLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fromCache := true
	country := r.URL.Query().Get("country")
	hash := r.URL.Query().Get("hash")
	regNr := r.URL.Query().Get("regnr")
	vin := r.URL.Query().Get("vin")
	regCountry := vehicle.RegCountryFromString(country) // For now, we're forcing unknown countries into "DK".
	if regNr == "" && vin == "" && hash == "" {
		srv.JSONError(w, APIError{http.StatusBadRequest, errLookup, "Missing query parameter 'hash', 'regnr' or 'vin'"})
		return
	}
	// "country" is required for "regnr" and "vin" only.
	if country == "" && hash == "" {
		srv.JSONError(w, APIError{http.StatusBadRequest, errLookup, "Missing query parameter 'country'"})
		return
	}
	var (
		veh vehicle.Vehicle
		err error
	)
	if hash != "" {
		veh, err = srv.store.LookupByHash(hash)
	} else if regNr != "" {
		veh, err = srv.store.LookupByRegNr(regCountry, regNr, false)
	} else {
		veh, err = srv.store.LookupByVIN(regCountry, vin, false)
	}
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errLookup, err.Error()})
		return
	}
	if veh == (vehicle.Vehicle{}) {
		// No cached result, so we attempt a direct lookup.
		mngr := srv.lookupMngr.FindServiceByCountry(regCountry)
		if mngr == nil {
			// No manager exists for that country, so let's give up.
			fmt.Printf("No direct lookups supported for country %s\n", vehicle.RegCountryToString(regCountry))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if regNr != "" {
			fmt.Printf("Cache miss: performing direct lookup of reg.nr. %s via %q\n", regNr, mngr.Name())
			veh, err = mngr.LookupRegNr(regNr)
		} else {
			fmt.Printf("Cache miss: performing direct lookup of VIN %s via %q\n", vin, mngr.Name())
			veh, err = mngr.LookupVIN(vin)
		}
		if err != nil {
			srv.JSONError(w, APIError{http.StatusInternalServerError, errLookup, err.Error()})
			return
		}
		// At this point, we found a vehicle via direct lookup. Let's cache it for future lookups!
		if err = veh.GenHash(); err != nil {
			// We just print the error. It doesn't prevent the request from completing.
			fmt.Printf("Error generating hash for vehicle with Ident %d\n", veh.MetaData.Ident)
		} else {
			if _, err := srv.store.SyncVehicle(veh); err != nil {
				// Again, we don't let this error interrupt the request.
				fmt.Printf("Unable to add vehicle with Ident %d\n", veh.MetaData.Ident)
			}
		}
		fromCache = false
	}
	bytes, err := json.Marshal(vehicleToAPIType(veh, fromCache))
	if err != nil {
		srv.JSONError(w, APIError{http.StatusInternalServerError, errMarshalling, err.Error()})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

// handleVehicle currently only allows enabling/disabling a specific vehicle by hash value.
func (srv *WebServer) handleVehicle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	hash := r.URL.Query().Get("hash")
	op := r.URL.Query().Get("op")
	if hash == "" || op == "" {
		srv.JSONError(w, APIError{http.StatusBadRequest, errVehicleOp, "Missing query parameter 'hash' and/or 'op'. Valid values for 'op': 'enable', 'disable'"})
		return
	}
	switch op {
	case "disable":
		if err := srv.store.Disable(hash); err != nil {
			if err == vehicle.ErrNoSuchVehicle {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			srv.JSONError(w, APIError{http.StatusInternalServerError, errVehicleOp, err.Error()})
			return
		}
	case "enable":
		if err := srv.store.Enable(hash); err != nil {
			if err == vehicle.ErrNoSuchVehicle {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			srv.JSONError(w, APIError{http.StatusInternalServerError, errVehicleOp, err.Error()})
			return
		}
	default:
		srv.JSONError(w, APIError{http.StatusBadRequest, errVehicleOp, fmt.Sprintf("No such operation: %s", op)})
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		sched := scheduler.New(srv.cnf, srv.store)
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
