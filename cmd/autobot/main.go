// Copyright 2018 Martin Kock.

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/OmniCar/autobot/args"
	"github.com/OmniCar/autobot/config"
	"github.com/OmniCar/autobot/dataprovider"
	"github.com/OmniCar/autobot/dmr"
	"github.com/OmniCar/autobot/vehiclestore"
)

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
	// Parse CLI arguments.
	args, err := args.ParseCLI()
	if err != nil {
		log.Fatalf("Failed to parse CLI arguments: %s", err)
	}
	if args.Debug {
		go monitorRuntime()
	}

	// Load and parse configuration file.
	if args.ConfigFile == "" {
		log.Fatalf("Need a configuration file")
	}
	cnf, err := config.NewConfig(args.ConfigFile)
	if err != nil {
		log.Fatalf("Unable to load configuration file %s", args.ConfigFile)
	}

	store := vehiclestore.NewVehicleStore(cnf.MemStore, cnf.Sync)
	if err := store.Open(); err != nil {
		log.Fatalf("Unable to connect to memory store, check your configuration file")
	}
	defer func() {
		store.Close()
	}()
	if args.VIN != "" {
		vehicle, err := store.LookupByVIN(args.VIN)
		if err != nil {
			fmt.Printf("Unable to lookup VIN %s: %s\n", args.VIN, err)
			os.Exit(0)
		}
		fmt.Println("Found!")
		fmt.Println(vehicle.FlexString("\n", "  "))
		os.Exit(0)
	} else if args.RegNr != "" {
		vehicle, err := store.LookupByRegNr(args.RegNr)
		if err != nil {
			fmt.Printf("Unable to lookup reg. nr. %s: %s\n", args.RegNr, err)
			os.Exit(0)
		}
		fmt.Println("Found!")
		fmt.Println(vehicle.FlexString("\n", "  "))
		os.Exit(0)
	} else if args.Clear {
		if err := store.Clear(); err != nil {
			log.Fatal("Unable clear store, manual cleanup required\n")
		}
		fmt.Println("Store cleared")
		os.Exit(0)
	} else if args.Status {
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
	if args.SourceFile == "" {
		fmt.Printf("Using FTP data file at %q\n", cnf.Ftp.Host)
		ptype = dataprovider.FtpProv
	} else {
		fmt.Printf("Using local data file: %s\n", args.SourceFile)
		ptype = dataprovider.FsProv
	}

	prov := dataprovider.NewProvider(ptype, cnf)
	src, err := prov.Provide(args.SourceFile)
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
