// Copyright 2018 Martin Kock.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/OmniCar/autobot/config"
	"github.com/OmniCar/autobot/dataprovider"
	"github.com/OmniCar/autobot/dmr"
	"github.com/OmniCar/autobot/vehiclestore"
)

var cnfFile = flag.String("cnffile", "config.toml", "Configuration file for FTP connectivity")
var inFile = flag.String("infile", "", "DMR XML file in UTF-8 format")

func main() {
	flag.Parse()

	// Load and parse configuration file.
	if *cnfFile == "" {
		log.Fatalf("need a configuration file")
	}
	cnf, err := config.NewConfig(*cnfFile)
	if err != nil {
		log.Fatalf("unable to load configuration file %s", *cnfFile)
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

	store := vehiclestore.NewVehicleStore(cnf.MemStore, cnf.Sync)
	if err := store.Open(); err != nil {
		log.Fatalf("unable to connect to memory store, check your configuration file")
	}
	id := store.NewSyncOp(dataprovider.ProvTypeString(ptype))

	dmrService := dmr.NewService()
	vehicles, done := dmrService.LoadNew(src)
	defer func() {
		store.Close()
	}()
	if err := store.Sync(id, vehicles, done); err != nil {
		log.Fatalf("error during sync: %s", err)
	}
	fmt.Println(store.Status(id))
}
