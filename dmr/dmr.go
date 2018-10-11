package dmr

import (
	"io"

	"github.com/OmniCar/autobot/autoservice"
)

// Service represents DMR (Danish Motor Registry).
type Service struct {
}

// NewService returns a service that satisfies the VehicleLoader interface.
func NewService() Service {
	return Service{}
}

func (dmr *Service) processFile(rc io.ReadCloser, vehicles chan<- autoservice.Vehicle, done chan<- bool) {

}

// LoadNew loads all new vehicles from DMR and returns them on a channel.
func (dmr *Service) LoadNew(rc io.ReadCloser) (vehicles chan<- autoservice.Vehicle, done chan<- bool, err error) {
	vehicles, done = make(chan autoservice.Vehicle), make(chan bool)
	go dmr.processFile(rc, vehicles, done)
	return vehicles, done, nil
}
