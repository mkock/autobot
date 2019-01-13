package webservice

import (
	"fmt"
	"net/http"

	"github.com/mkock/autobot/vehicle"
)

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
