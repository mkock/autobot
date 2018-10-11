package autoservice

import (
	"fmt"
	"time"
)

// VehicleList contains vehicles that were found during parsing.
type VehicleList map[uint64]Vehicle

// Meta contains metadata for each vehicle.
type Meta struct {
	Source      string
	Ident       uint64
	LastUpdated time.Time
}

// Vehicle contains the core vehicle data that Autobot manages.
type Vehicle struct {
	MetaData     Meta
	RegNr        string
	VIN          string
	Brand        string
	Model        string
	FuelType     string
	FirstRegDate time.Time
}

// String returns a stringified representation of the Vehicle data structure.
func (v Vehicle) String() string {
	return fmt.Sprintf("%#v", v)
}

// VehicleLoader is the interface that each service must satisfy in order to provide vehicle data.
type VehicleLoader interface {
	HasNew() (bool, error)
	LoadNew() (vehicles chan<- Vehicle, done chan<- bool, err error)
}
