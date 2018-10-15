// Copyright 2018 Martin Kock.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/OmniCar/autobot"
	"github.com/OmniCar/autobot/autoservice"
	"github.com/OmniCar/autobot/dataprovider"
	"github.com/OmniCar/autobot/dmr"
)

var cnfFile = flag.String("cnffile", "config.toml", "Configuration file for FTP connectivity")
var inFile = flag.String("infile", "", "DMR XML file in UTF-8 format")
var outFile = flag.String("outfile", "out.csv", "Name of file to stream CSV data to")

func main() {
	flag.Parse()

	// Load and parse configuration file.
	if *cnfFile == "" {
		log.Fatalf("Autobot: need a configuration file")
	}
	cnf, err := autobot.NewConfig(*cnfFile)
	if err != nil {
		log.Fatalf("Autobot: Unable to load configuration file %s", *cnfFile)
	}

	var src io.ReadCloser
	if *inFile == "" {
		fmt.Printf("Using FTP data file at %q\n", cnf.Ftp.Host)
		prov := dataprovider.NewFtpProvider(cnf.Ftp)
		if err := prov.Open(*inFile); err != nil {
			log.Fatalf("Autobot: %s", err)
		}
		fname, _ := prov.CheckForLatest()
		if fname == "" {
			fmt.Println("No new stat files detected.")
			return
		}
		fmt.Println("New stat file detected: " + fname)
		fmt.Println("Fetching...")
		if src, err = prov.Provide(); err != nil {
			log.Fatalf("Autobot: %s", err)
		}
		prov.Close()
	} else {
		fmt.Printf("Using local data file: %s\n", *inFile)
		prov := dataprovider.NewFileProvider()
		if err := prov.Open(*inFile); err != nil {
			log.Fatalf("Autobot: %s", err)
		}
		if src, err = prov.Provide(); err != nil {
			log.Fatal(err)
		}
		prov.Close()
	}

	// Instantiate an XML parser.
	parser := dmr.NewXMLParser()

	// Nr. of workers = cpu core count - 1 for the main go routine.
	numWorkers := int(math.Max(1.0, float64(runtime.NumCPU()-1)))

	// Prepare channels for communicating parsed data and termination.
	lines, parsed, done := make(chan []string, numWorkers), make(chan autoservice.Vehicle, numWorkers), make(chan int)

	// Start the number of workers (parsers) determined by numWorkers.
	fmt.Printf("Starting %v workers...\n", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go parser.ParseExcerpt(i, lines, parsed, done)
	}

	// Main file scanner go routine.
	go func() {
		scanner := bufio.NewScanner(src)
		excerpt := []string{}
		grab := false
		defer func() {
			close(lines)
			src.Close()
		}()
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
	}()

	var vehicles autoservice.VehicleList = make(map[uint64]autoservice.Vehicle) // For keeping track of unique vehicles.

	// Wait for parsed excerpts to come in, and ensure their uniqueness by using a map.
	waits := numWorkers
	for {
		select {
		case vehicle := <-parsed:
			if _, ok := vehicles[vehicle.MetaData.Ident]; !ok {
				vehicles[vehicle.MetaData.Ident] = vehicle
			}
		case <-done:
			waits--
			if waits == 0 {
				writeToFile(vehicles, *outFile)
				if *inFile != "" {
					// Only log processed files from the FTP.
					autobot.LogAsProcessed(*inFile)
				}
				return
			}
		}
	}

}

func writeToFile(vehicles autoservice.VehicleList, outFile string) {
	out, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Unable to open output file %v for writing.\n", outFile)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	// Write CSV data.
	for _, vehicle := range vehicles {
		_, err := out.WriteString(vehicle.String() + "\n")
		if err != nil {
			fmt.Println("Unable to write to output file, unknown write error")
		}
	}

	fmt.Printf("Done - CSV data written to %q\n", outFile)
}
