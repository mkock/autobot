package dmr

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"

	"github.com/OmniCar/autobot/autoservice"
)

// Service represents DMR (Danish Motor Registry).
type Service struct {
}

// NewService returns a service that satisfies the VehicleLoader interface.
func NewService() *Service {
	return &Service{}
}

// processFile takes a file handle to an open XML file, and starts up "numWorkers" workers that will parse each XML
// excerpt concurrently while delivering the parsed vehicles on the "vehicles" channel. It will send the worker id on
// the "done" channel for each worker when parsing has completed.
func (service *Service) processFile(rc io.ReadCloser, numWorkers int, vehicles chan<- autoservice.Vehicle, done chan<- int) {
	// Instantiate an XML parser.
	parser := NewXMLParser()
	lines := make(chan []string, numWorkers)

	// Start the number of workers (parsers) determined by numWorkers.
	fmt.Printf("Starting %v workers...\n", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go parser.ParseExcerpt(i, lines, vehicles, done)
	}

	// Preparations for the main loop.
	scanner := bufio.NewScanner(rc)
	excerpt := []string{}
	grab := false
	defer func() {
		close(lines)
		rc.Close()
	}()

	// Main file scanner go routine.
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "<ns:Statistik>") {
			grab = true
		} else if strings.HasPrefix(line, "</ns:Statistik>") {
			grab = false
			excerpt = append(excerpt, line)
			lines <- excerpt // On every closing elem. we send the excerpt to a worker and move on.
			excerpt = nil
		}
		if grab {
			excerpt = append(excerpt, line)
		}
	}
}

// LoadNew loads all new vehicles from DMR and returns them on a channel.
// It will send True on channel "done" once all vehicles have been processed.
func (service *Service) LoadNew(rc io.ReadCloser) (vehicles chan autoservice.Vehicle, done chan bool) {
	// Nr. of workers = cpu core count - 1 for the main go routine.
	numWorkers := int(math.Max(1.0, float64(runtime.NumCPU()-1)))
	vehicles, done = make(chan autoservice.Vehicle), make(chan bool)
	workerDone := make(chan int, numWorkers)
	go service.processFile(rc, numWorkers, vehicles, workerDone)

	// Collect answers from individual workers and send True on "done".
	go func() {
		for i := 0; i < numWorkers; i++ {
			_ = <-workerDone
			// fmt.Printf("Worker %d is done.", wid)
		}
		done <- true
	}()

	return vehicles, done
}
