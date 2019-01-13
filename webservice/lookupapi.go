package webservice

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mkock/autobot/vehicle"
)

// handleLookup allows vehicle lookups based on hash value, VIN or registration number. A country must always be
// provided.
// @TODO: There is too much business logic here; put it somewhere else.
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
			fmt.Printf("No direct lookups supported for country %s\n", regCountry.String())
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
