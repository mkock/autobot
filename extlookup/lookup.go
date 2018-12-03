package extlookup

import "github.com/mkock/autobot/vehicle"

// Lookupable is the interface that each direct vehicle lookup service must implement.
type Lookupable interface {
	Name() string
	Supports(country vehicle.RegCountry) bool
	LookupVIN(string) (vehicle.Vehicle, error)
	LookupRegNr(string) (vehicle.Vehicle, error)
}

// Manager keeps track of, and lets you interact with, registered lookup services.
type Manager map[string]Lookupable

// NewManager returns a new Manager, which is able to keep track of multiple lookup services.
func NewManager() *Manager {
	return &Manager{}
}

// contains returns true if the Manager contains a lookup service with the given name.
func (mngr *Manager) contains(name string) bool {
	_, ok := (*mngr)[name]
	return ok
}

// AddService adds an auto service to the Manager.
func (mngr *Manager) AddService(service Lookupable) {
	if mngr.contains(service.Name()) {
		return
	}
	(*mngr)[service.Name()] = service
}

// FindServiceByCountry returns the first service that supports the given country, or nil.
func (mngr *Manager) FindServiceByCountry(country vehicle.RegCountry) Lookupable {
	for _, service := range *mngr {
		if service.Supports(country) {
			return service
		}
	}
	return nil
}
