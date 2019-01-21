package vehicle

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"
)

// List contains vehicles that were found during parsing.
type List map[uint64]Vehicle

// RegCountry represents a country of registration for a vehicle.
type RegCountry int

// List of allowed registration countries.
const (
	DK RegCountry = iota
	NO
)

// String returns the string representation of the RegCountry.
func (reg RegCountry) String() string {
	for key, val := range regCountryMap {
		if val == reg {
			return key
		}
	}
	return "DK" // Default.
}

// Type represents the overall type of vehicle, ie. car, trailer, van etc.
type Type int

// List of supported vehicle types.
const (
	Unknown Type = iota
	Car
	Bus
	Van
	Truck
	Trailer
)

// String returns the string representation of the vehicle type.
func (t Type) String() string {
	switch t {
	case Car:
		return "Car"
	case Bus:
		return "Bus"
	case Van:
		return "Van"
	case Truck:
		return "Truck"
	case Trailer:
		return "Trailer"
	default:
		return "Unknown"
	}
}

// TypeFromString returns the Type that matches the given string (case insensitive match).
// If TypeFromString does not find a direct match, Type.Unknown is returned.
func TypeFromString(str string) Type {
	switch strings.ToLower(str) {
	case "car":
		return Car
	case "bus":
		return Bus
	case "van":
		return Van
	case "truck":
		return Truck
	case "trailer":
		return Trailer
	default:
		return Unknown
	}
}

// Meta contains metadata for each vehicle.
type Meta struct {
	Hash        uint64
	Source      string
	Country     RegCountry
	Ident       uint64
	LastUpdated time.Time
	Disabled    bool
}

// Vehicle contains the core vehicle data that Autobot manages.
// As vehicles are persisted in Redis / Google Memory Store, they should not contain pointers.
type Vehicle struct {
	MetaData     Meta `hash:"ignore"`
	Type         Type
	RegNr        string
	VIN          string
	Brand        string
	Model        string
	FuelType     string
	Variant      string
	FirstRegDate time.Time
}

// Marshal converts the given Vehicle to a string using JSON encoding.
func (v *Vehicle) Marshal() (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Unmarshal converts the given string to a Vehicle using JSON decoding.
func (v *Vehicle) Unmarshal(str string) error {
	return json.Unmarshal([]byte(str), v)
}

var regCountryMap = map[string]RegCountry{
	"DK": DK,
	"NO": NO,
}

// RegCountryFromString takes a string and returns the matching country of registration.
func RegCountryFromString(reg string) RegCountry {
	elem, ok := regCountryMap[reg]
	if ok {
		return elem
	}
	return DK // Default.
}

// GenHash generates a unique hash value of the vehicle. The hash is stored in the vehicle metadata.
func (v *Vehicle) GenHash() error {
	hash, err := hashstructure.Hash(v, nil)
	if err != nil {
		return fmt.Errorf("unable to hash Vehicle with Ident: %d", v.MetaData.Ident)
	}
	v.MetaData.Hash = hash
	return nil
}

// String returns a stringified representation of the Vehicle data structure.
func (v Vehicle) String() string {
	return v.FlexString("", " ")
}

// FlexString returns a stringified multi-line representation of the Vehicle data structure.
func (v Vehicle) FlexString(lb, leftPad string) string {
	var txt strings.Builder
	fmt.Fprintf(&txt, "#%d (%s)%s", v.MetaData.Hash, DisabledAsString(v.MetaData.Disabled), lb)
	fmt.Fprintf(&txt, "%sCountry: %s%s", leftPad, v.MetaData.Country.String(), lb)
	fmt.Fprintf(&txt, "%sIdent: %d%s", leftPad, v.MetaData.Ident, lb)
	fmt.Fprintf(&txt, "%sRegNr: %s%s", leftPad, v.RegNr, lb)
	fmt.Fprintf(&txt, "%sVIN: %s%s", leftPad, v.VIN, lb)
	fmt.Fprintf(&txt, "%sBrand: %s%s", leftPad, v.Brand, lb)
	fmt.Fprintf(&txt, "%sModel: %s%s", leftPad, v.Model, lb)
	fmt.Fprintf(&txt, "%sVariant: %s%s", leftPad, v.Variant, lb)
	fmt.Fprintf(&txt, "%sFuelType: %s%s", leftPad, v.FuelType, lb)
	fmt.Fprintf(&txt, "%sRegDate: %s%s", leftPad, v.FirstRegDate.Format("2006-01-02"), lb)
	return txt.String()
}

// Slice returns most properties from Vehicle as a slice of strings, intended for use in CSV conversions.
func (v Vehicle) Slice() [10]string {
	hash := strconv.FormatUint(v.MetaData.Hash, 10)
	country := v.MetaData.Country.String()
	ident := strconv.FormatUint(v.MetaData.Ident, 10)
	firstReg := v.FirstRegDate.Format("2006-01-02")
	props := [10]string{hash, country, ident, v.RegNr, v.VIN, v.Brand, v.Model, v.Variant, v.FuelType, firstReg}
	return props
}

// PrettyBrandName titles-cases the given brand name unless its length is 3 or below, in which case everything is
// uppercased. This should handle most cases.
func PrettyBrandName(brand string) string {
	if len(brand) <= 3 {
		return strings.ToUpper(brand)
	}
	return strings.Title(strings.ToLower(brand))
}

// PrettyFuelType normalizes fuel-type by capitalizing the first letter only.
func PrettyFuelType(ft string) string {
	return strings.Title(strings.ToLower(ft))
}

// HashAsKey converts the given hash value into a string that can be used as key in the vehicle store.
func HashAsKey(hash uint64) string {
	return strconv.FormatUint(hash, 10)
}

// DisabledAsString returns a stringified version of the Disabled field.
func DisabledAsString(status bool) string {
	if status {
		return "Disabled"
	}
	return "Active"
}
