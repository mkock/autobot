package autoservice

import (
	"fmt"
	"strings"
	"time"
)

// VehicleList contains vehicles that were found during parsing.
type VehicleList map[uint64]Vehicle

// Meta contains metadata for each vehicle.
type Meta struct {
	Hash        uint64
	Source      string
	Ident       uint64
	LastUpdated time.Time
}

// Vehicle contains the core vehicle data that Autobot manages.
// As vehicles are persisted in Redis / Google Memory Store, they should not contain pointers.
type Vehicle struct {
	MetaData     Meta `hash:"ignore"`
	RegNr        string
	VIN          string
	Brand        string
	Model        string
	FuelType     string
	FirstRegDate time.Time
}

// String returns a stringified representation of the Vehicle data structure.
func (v Vehicle) String() string {
	return v.FlexString("", " ")
}

// FlexString returns a stringified multi-line representation of the Vehicle data structure.
func (v Vehicle) FlexString(lb, leftPad string) string {
	var txt strings.Builder
	fmt.Fprintf(&txt, "#%d%s", v.MetaData.Hash, lb)
	fmt.Fprintf(&txt, "%sIdent: %d%s", leftPad, v.MetaData.Ident, lb)
	fmt.Fprintf(&txt, "%sRegNr: %s%s", leftPad, v.RegNr, lb)
	fmt.Fprintf(&txt, "%sVIN: %s%s", leftPad, v.VIN, lb)
	fmt.Fprintf(&txt, "%sBrand: %s%s", leftPad, v.Brand, lb)
	fmt.Fprintf(&txt, "%sModel: %s%s", leftPad, v.Model, lb)
	fmt.Fprintf(&txt, "%sFuelType: %s%s", leftPad, v.FuelType, lb)
	fmt.Fprintf(&txt, "%sRegDate: %s%s", leftPad, v.FirstRegDate.Format("2006-01-02"), lb)
	return txt.String()
}

// VehicleLoader is the interface that each service must satisfy in order to provide vehicle data.
type VehicleLoader interface {
	HasNew() (bool, error)
	LoadNew() (vehicles chan<- Vehicle, done chan<- bool)
}
