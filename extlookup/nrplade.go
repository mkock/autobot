package extlookup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/vehicle"
)

// NewNrpladeService returns a new Nrplade service for direct vehicle lookups.
func NewNrpladeService(cnf config.LookupConfig) *NrpladeService {
	return &NrpladeService{Service{"Nrplade", cnf}}
}

// NrpladeService integrates with a Danish license plate lookup service.
type NrpladeService struct {
	Service
}

// nrpladeData represents the HTTP response from nrpla.de.
type nrPladeData struct {
	Data nrpladeResponse `json:"data"`
}

// nrpladeResponse represents the main HTTP response (inside "data") from nrpla.de.
type nrpladeResponse struct {
	Registration string `json:"registration"`
	FirstRegDate string `json:"first_registration_date"`
	Vin          string `json:"vin"`
	Type         string `json:"type"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Version      string `json:"version"`
	FuelType     string `json:"fuel_type"`
	RegStatus    string `json:"registration_status"`
}

// decodeNrpladeBody does the decoding of the HTTP response into a vehicle.Vehicle.
func decodeNrpladeBody(body []byte) (vehicle.Vehicle, error) {
	reader := bytes.NewReader(body)
	decoder := json.NewDecoder(reader)
	data := &nrPladeData{}
	err := decoder.Decode(data)
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	regDate, err := time.Parse("2006-01-02", data.Data.FirstRegDate)
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	vehicle := vehicle.Vehicle{
		MetaData:     vehicle.Meta{},
		RegNr:        data.Data.Registration,
		VIN:          data.Data.Vin,
		Brand:        data.Data.Brand,
		Model:        data.Data.Model,
		FuelType:     data.Data.FuelType,
		FirstRegDate: regDate,
	}
	return vehicle, nil
}

// LookupRegNr looks up a vehicle based on registration number.
func (service *NrpladeService) LookupRegNr(regNr string) (vehicle.Vehicle, error) {
	proto := "http"
	if service.Conf.LookupSecure {
		proto = proto + "s" // https
	}
	reqURL, err := url.Parse(fmt.Sprintf("%s://%s/%s/%s?api_token=%s", proto, service.Conf.LookupHost, service.Conf.LookupPath, regNr, service.Conf.LookupKey))
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	body, err := service.makeReq(reqURL.String())
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	return decodeNrpladeBody(body)
}

// LookupVIN looks up a vehicle based on VIN number.
func (service *NrpladeService) LookupVIN(vin string) (vehicle.Vehicle, error) {
	proto := "http"
	if service.Conf.LookupSecure {
		proto = proto + "s" // https
	}
	reqURL, err := url.Parse(fmt.Sprintf("%s://%s/%s/vin/%s?api_token=%s", proto, service.Conf.LookupHost, service.Conf.LookupPath, vin, service.Conf.LookupKey))
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	body, err := service.makeReq(reqURL.String())
	if err != nil {
		return vehicle.Vehicle{}, err
	}
	return decodeNrpladeBody(body)
}

// Name returns the service name.
func (service *NrpladeService) Name() string {
	return service.Service.Name
}

// Supports returns true if the given country is supported by this service.
func (service *NrpladeService) Supports(country vehicle.RegCountry) bool {
	return country == vehicle.DK
}
