// Copyright 2018 Martin Kock.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/OmniCar/autobot/config"
	"github.com/OmniCar/autobot/dataprovider"
	"github.com/OmniCar/autobot/dmr"
	"github.com/OmniCar/autobot/vehiclestore"
)

var cnfFile = flag.String("cnffile", "config.toml", "Configuration file for FTP connectivity")
var inFile = flag.String("infile", "", "DMR XML file in UTF-8 format")
var vin = flag.String("vin", "", "VIN number to lookup, if any (will not synchronize data)")
var regNr = flag.String("regnr", "", "Registration number to lookup, if any (will not synchronize data)")
var debug = flag.Bool("debug", false, "Print CPU count, goroutine count and memory usage every 10 seconds")
var clear = flag.Bool("clear", false, "Clears the entire vehicle store")
var status = flag.Bool("status", false, "Displays the last synchronisation log")

func monitorRuntime() {
	log.Println("Number of CPUs:", runtime.NumCPU())
	m := &runtime.MemStats{}
	for {
		r := runtime.NumGoroutine()
		log.Println("Number of goroutines", r)
		runtime.ReadMemStats(m)
		log.Println("Allocated memory:", m.Alloc)
		time.Sleep(10 * time.Second)
	}
}

func main() {
	if *debug {
		go monitorRuntime()
	}
	flag.Parse()

	// Load and parse configuration file.
	if *cnfFile == "" {
		log.Fatalf("need a configuration file")
	}
	cnf, err := config.NewConfig(*cnfFile)
	if err != nil {
		log.Fatalf("unable to load configuration file %s", *cnfFile)
	}

	store := vehiclestore.NewVehicleStore(cnf.MemStore, cnf.Sync)
	if err := store.Open(); err != nil {
		log.Fatalf("unable to connect to memory store, check your configuration file")
	}
	defer func() {
		store.Close()
	}()
	if *vin != "" {
		vehicle, err := store.LookupByVIN(*vin)
		if err != nil {
			fmt.Printf("Unable to lookup VIN %s: %s\n", *vin, err)
			os.Exit(0)
		}
		fmt.Println("Found!")
		fmt.Println(vehicle.FlexString("\n", "  "))
		os.Exit(0)
	} else if *regNr != "" {
		vehicle, err := store.LookupByRegNr(*regNr)
		if err != nil {
			fmt.Printf("Unable to lookup reg. nr. %s: %s\n", *regNr, err)
			os.Exit(0)
		}
		fmt.Println("Found!")
		fmt.Println(vehicle.FlexString("\n", "  "))
		os.Exit(0)
	} else if *clear {
		if err := store.Clear(); err != nil {
			log.Fatal("Unable clear store, manual cleanup required\n")
		}
		fmt.Println("Store cleared")
		os.Exit(0)
	} else if *status {
		entries, err := store.CountLog()
		if err != nil {
			log.Fatalf("Error while fetching status: %s", err)
		}
		fmt.Printf("Status: %d log entries in total. Last entry:\n", entries)
		entry, err := store.LastLog()
		if err != nil {
			log.Fatalf("Error while fetching status: %s", err)
		}
		fmt.Println(entry.String())
		os.Exit(0)
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
	if err != nil {
		log.Fatalf("error during file retrieval: %s", err)
	}
	if src == nil {
		log.Print("No stat file detected. Quitting.")
		os.Exit(0)
	}

	id := store.NewSyncOp(dataprovider.ProvTypeString(ptype))

	dmrService := dmr.NewService()
	vehicles, done := dmrService.LoadNew(src)
	if err := store.Sync(id, vehicles, done); err != nil {
		log.Fatalf("error during sync: %s", err)
	}
	fmt.Println(store.Status(id))
}
