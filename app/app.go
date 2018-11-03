package app

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/OmniCar/autobot/config"
	"github.com/OmniCar/autobot/dataprovider"
	"github.com/OmniCar/autobot/dmr"
	"github.com/OmniCar/autobot/vehicle"
	"github.com/OmniCar/autobot/webservice"
	"github.com/jessevdk/go-flags"
)

var (
	parser     *flags.Parser  // Initialised in init.
	globalOpts *Options       // Initialised in init.
	conf       config.Config  // Initialised in bootstrap.
	store      *vehicle.Store // Initialised in bootstrap.
)

// init (called automatically) sets up the CLI parser.
func init() {
	// Setup commands.
	var (
		syncCmd    SyncCommand
		clearCmd   ClearCommand
		statusCmd  StatusCommand
		lookupCmd  LookupCommand
		disableCmd DisableCommand
		serveCmd   ServeCommand
	)
	globalOpts = &Options{}
	parser = flags.NewParser(globalOpts, flags.Default)
	parser.CommandHandler = bootstrap // Loads conf and connects to the vehicle store before executing commands.
	parser.AddCommand("sync", "synchronise", "synchronise the vehicle store with an external data source", &syncCmd)
	parser.AddCommand("clear", "clear", "clears the vehicle store of all vehicles", &clearCmd)
	parser.AddCommand("status", "status", "displays a short status of the vehicle store", &statusCmd)
	parser.AddCommand("lookup", "vehicle lookup", "performs a vehicle lookup, by VIN or registration number", &lookupCmd)
	parser.AddCommand("disable", "vehicle disabling", "disables a vehicle so it won't appear in lookups", &disableCmd)
	parser.AddCommand("serve", "serve", "starts autobot as a web server", &serveCmd)
}

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

// Options contains command-line arguments parsed upon application initialisation.
type Options struct {
	ConfigFile string `short:"c" long:"config-file" required:"yes" default:"config.toml" description:"Application configuration file in TOML format"`
}

// ServeCommand is responsible for initialising and booting up a web server that supports much of the same functionality
// that is also provided via the CLI.
type ServeCommand struct {
	Port uint `short:"p" long:"port" default:"1826" description:"Port number to listen on, defaults to 1826"`
}

// Execute runs the web server. It does not return unless the web server stops functioning.
func (cmd *ServeCommand) Execute(opts []string) error {
	api := webservice.New()
	fmt.Printf("Serving on port %d\n", cmd.Port)
	if err := api.Serve(cmd.Port); err != nil {
		return err
	}
	return nil
}

// SyncCommand contains options for synchronising the vehicle store with an external source.
type SyncCommand struct {
	SourceFile string `short:"f" long:"source-file" description:"DMR XML file in UTF-8 format"`
	Debug      bool   `short:"d" long:"debug" description:"Debug: print CPU count, goroutine count and memory usage every 10 seconds"`
}

// Execute runs the command.
func (cmd *SyncCommand) Execute(opts []string) error {
	if cmd.Debug {
		go monitorRuntime()
	}
	var ptype int
	if cmd.SourceFile == "" {
		fmt.Printf("Using FTP data file at %q\n", conf.Ftp.Host)
		ptype = dataprovider.FtpProv
	} else {
		fmt.Printf("Using local data file: %s\n", cmd.SourceFile)
		ptype = dataprovider.FsProv
	}

	prov := dataprovider.NewProvider(ptype, conf)
	src, err := prov.Provide(cmd.SourceFile)
	if err != nil {
		return err
	}
	if src == nil {
		log.Print("No stat file detected. Aborting.")
		return nil
	}

	id := store.NewSyncOp(dataprovider.ProvTypeString(ptype))

	dmrService := dmr.NewService()
	vehicles, done := dmrService.LoadNew(src)
	if err := store.Sync(id, vehicles, done); err != nil {
		return err
	}
	fmt.Println(store.Status(id))

	return nil
}

// ClearCommand contains options for clearing the vehicle store.
type ClearCommand struct {
	Clear bool `short:"l" long:"clear" description:"Clears the entire vehicle store"`
}

// Execute runs the command.
func (cmd *ClearCommand) Execute(opts []string) error {
	if err := store.Clear(); err != nil {
		return err
	}
	return nil
}

// StatusCommand contains options for displaying the status of the vehicle store.
type StatusCommand struct {
	Status bool `short:"s" long:"status" description:"Displays the last synchronisation log"`
}

// Execute runs the command.
func (cmd *StatusCommand) Execute(opts []string) error {
	entries, err := store.CountLog()
	if err != nil {
		return err
	}
	fmt.Printf("Status: %d log entries in total. Last entry:\n", entries)
	entry, err := store.LastLog()
	if err != nil {
		return err
	}
	fmt.Println(entry.String())
	return nil
}

// LookupCommand contains options for vehicle lookups using reg.nr. or VIN.
type LookupCommand struct {
	Country  string `short:"c" long:"country" description:"Country where vehicle is registered" required:"yes" choice:"DK" choice:"NO"`
	VIN      string `short:"v" long:"vin" description:"VIN number to lookup, if any (will not synchronize data)"`
	RegNr    string `short:"r" long:"regnr" description:"Registration number to lookup, if any (will not synchronize data)"`
	Disabled bool   `short:"d" long:"disabled" description:"Include vehicle in result even if disabled"`
}

// Execute is called by go-flags and thus bootstraps the LookupCommand.
func (cmd *LookupCommand) Execute(opts []string) error {
	var (
		nr     string
		desc   string
		lookup func(vehicle.RegCountry, string, bool) (vehicle.Vehicle, error)
	)
	if cmd.RegNr != "" {
		nr = cmd.RegNr
		desc = "registration number"
		lookup = store.LookupByRegNr
	} else if cmd.VIN != "" {
		lookup = store.LookupByVIN
		desc = "VIN"
		nr = cmd.VIN
	} else {
		fmt.Println("Lookup: need VIN or registration number")
		return nil
	}
	veh, err := lookup(vehicle.RegCountryFromString(cmd.Country), nr, cmd.Disabled)
	if err != nil {
		return err
	}
	if veh == (vehicle.Vehicle{}) {
		fmt.Printf("No vehicle found with %s %s\n", desc, nr)
		return nil
	}
	fmt.Println("Found!")
	fmt.Println(veh.FlexString("\n", "  "))
	return nil
}

// DisableCommand disables a vehicle so it won't appear in lookups.
type DisableCommand struct {
	Hash string `short:"h" long:"hash" description:"Hash of vehicle to disable"`
}

// Execute runs the disable command.
func (cmd *DisableCommand) Execute(opts []string) error {
	return store.Disable(cmd.Hash)
}

// loadConfig loads the TOML configuration file with the specified name.
func loadConfig(fname string) (config.Config, error) {
	return config.NewConfig(fname)
}

// bootstrap loads the app configuration and connects to the vehicle store.
func bootstrap(cmd flags.Commander, args []string) error {
	// Load the configuration file passed via the CLI.
	var err error
	conf, err = loadConfig(globalOpts.ConfigFile)
	if err != nil {
		return err
	}

	// Connect to the vehicle store.
	store = vehicle.NewStore(conf.MemStore, conf.Sync)
	if err := store.Open(); err != nil {
		return err
	}
	defer func() {
		store.Close()
	}()

	// Carry on with command execution.
	return cmd.Execute(args)
}

// Start runs a command.
func Start() error {
	// This will run the command that matches the command-line options.
	if _, err := parser.Parse(); err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			return nil // This is to avoid help messages from being printed twice.
		}
		return err
	}
	return nil
}
