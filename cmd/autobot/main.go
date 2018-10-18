// Copyright 2018 Martin Kock.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/OmniCar/autobot"
	"github.com/OmniCar/autobot/autoservice"
	"github.com/OmniCar/autobot/config"
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
	cnf, err := config.NewConfig(*cnfFile)
	if err != nil {
		log.Fatalf("Autobot: Unable to load configuration file %s", *cnfFile)
	}

	var ptype int
	if *inFile == "" {
		fmt.Printf("Using FTP data file at %q\n", cnf.Ftp.Host)
		ptype = dataprovider.FtpProv
	} else {
		fmt.Printf("Using local data file: %s\n", *inFile)
		ptype = dataprovider.FsProv
	}

	prov := dataprovider.NewProvider(ptype, cnf)
	src, err := prov.Provide(*inFile)

	dmrService := dmr.NewService()
	vehicles, done := dmrService.LoadNew(src)
	var vlist autoservice.VehicleList = make(map[uint64]autoservice.Vehicle) // For keeping track of vehicles.

	// Wait for parsed excerpts to come in, and ensure their uniqueness by using a map.
	waits := cap(done)
Mainloop:
	for {
		select {
		case vehicle := <-vehicles:
			if _, ok := vlist[vehicle.MetaData.Ident]; !ok {
				vlist[vehicle.MetaData.Ident] = vehicle
			}
		case <-done:
			waits--
			if waits == 0 {
				writeToFile(vlist, *outFile)
				if *inFile != "" {
					// Only log processed files from the FTP.
					autobot.LogAsProcessed(*inFile)
				}
				break Mainloop
			}
		}
	}
	fmt.Printf("Done - CSV data written to %q\n", *outFile)
}

// This is temporary.
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
}
