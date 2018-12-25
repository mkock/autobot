package app

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/dataprovider"
	"github.com/mkock/autobot/dmr"
	"github.com/mkock/autobot/extlookup"
	"github.com/mkock/autobot/vehicle"
	"github.com/mkock/autobot/webservice"
)

var (
	version       = "1.0"            // Edited manually.
	parser        *flags.Parser      // Initialised in init.
	globalOpts    *Options           // Initialised in init.
	conf          config.Config      // Initialised in bootstrap.
	store         *vehicle.Store     // Initialised in bootstrap.
	lookupManager *extlookup.Manager // Initalised in bootstrap.
)

// init (called automatically) sets up the CLI parser.
func init() {
	// Setup commands.
	var (
		initCmd    InitCommand
		syncCmd    SyncCommand
		clearCmd   ClearCommand
		statusCmd  StatusCommand
		lookupCmd  LookupCommand
		disableCmd DisableCommand
		enableCmd  EnableCommand
		serveCmd   ServeCommand
		queryCmd   QueryCommand
		verCmd     VersionCommand
	)
	globalOpts = &Options{}
	parser = flags.NewParser(globalOpts, flags.Default)
	parser.CommandHandler = bootstrap // Loads conf and connects to the vehicle store before executing commands.
	parser.AddCommand("init", "initialise", "write an empty configuration file that you can fill out", &initCmd)
	parser.AddCommand("sync", "synchronise", "synchronise the vehicle store with an external data source", &syncCmd)
	parser.AddCommand("clear", "clear", "clears the vehicle store of all vehicles", &clearCmd)
	parser.AddCommand("status", "status", "displays a short status of the vehicle store", &statusCmd)
	parser.AddCommand("lookup", "lookup vehicle", "performs a vehicle lookup, by VIN or registration number", &lookupCmd)
	parser.AddCommand("disable", "disable vehicle", "disables a vehicle so it won't appear in lookups", &disableCmd)
	parser.AddCommand("enable", "enable vehicle", "enables a vehicle so it will reappear in lookups", &enableCmd)
	parser.AddCommand("serve", "serve", "starts autobot as a web server", &serveCmd)
	parser.AddCommand("query", "query vehicle store", "queries/searches the vehicle store using filters and selectors", &queryCmd)
	parser.AddCommand("version", "version", "displays the current build version of autobot", &verCmd)
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
	ConfigFile string `short:"c" long:"config-file" required:"no" default:"config.toml" description:"Application configuration file in TOML format"`
}

// InitCommand writes an empty configuration file to cwd, to help the user getting started with autobot.
type InitCommand struct{}

// Usage prints help text to the user.
func (cmd *InitCommand) Usage() string {
	return InitUsage
}

// Execute writes the empty configuration file to the current working directory.
func (cmd *InitCommand) Execute(opts []string) error {
	if err := config.WriteEmptyConf("config.toml"); err != nil {
		return err
	}
	fmt.Println("Wrote config.toml")
	return nil
}

// VersionCommand displays the current build version of autobot.
type VersionCommand struct{}

// Usage prints help text to the user.
func (cmd *VersionCommand) Usage() string {
	return VersionUsage
}

// Execute displays the current build version of autobot.
func (cmd *VersionCommand) Execute(opts []string) error {
	fmt.Printf("Autobot version: %s\n", version)
	return nil
}

// ServeCommand is responsible for initialising and booting up a web server that supports much of the same functionality
// that is also provided via the CLI.
type ServeCommand struct {
	Port   uint `short:"p" long:"port" default:"1826" description:"Port number to listen on"`
	NoSync bool `short:"s" long:"no-sync" description:"Runs the web server without background synchronisation"`
}

// Usage prints help text to the user.
func (cmd *ServeCommand) Usage() string {
	return ServeUsage
}

// Execute runs the web server. It does not return unless the web server stops functioning.
func (cmd *ServeCommand) Execute(opts []string) error {
	api := webservice.New(store, lookupManager, conf)
	log.Printf("Serving on port %d\n", cmd.Port)
	if err := api.Serve(cmd.Port, !cmd.NoSync); err != nil {
		return err
	}
	return nil
}

// SyncCommand contains options for synchronising the vehicle store with an external source.
type SyncCommand struct {
	Provider   string `short:"p" long:"provider" required:"yes" description:"Name of provider to sync with"`
	SourceFile string `short:"f" long:"source-file" description:"DMR XML file in UTF-8 format"`
	Debug      bool   `short:"d" long:"debug" description:"Debug: print CPU count, goroutine count and memory usage every 10 seconds"`
}

// Usage prints help text to the user.
func (cmd *SyncCommand) Usage() string {
	return SyncUsage
}

// Execute runs the command.
func (cmd *SyncCommand) Execute(opts []string) error {
	if cmd.Debug {
		go monitorRuntime()
	}
	var (
		ptype   int
		ok      bool
		provCnf config.ProviderConfig
	)
	if provCnf, ok = conf.Providers[cmd.Provider]; !ok {
		return fmt.Errorf("No such provider: %s", cmd.Provider)
	}
	if cmd.SourceFile == "" {
		log.Printf("Using FTP data file at %q\n", provCnf.Host)
		ptype = dataprovider.FtpProv
	} else {
		log.Printf("Using local data file: %s\n", cmd.SourceFile)
		ptype = dataprovider.FsProv
	}
	prov := dataprovider.NewProvider(ptype, provCnf)
	if err := prov.Open(); err != nil {
		return err
	}
	fname, err := prov.CheckForLatest(cmd.SourceFile)
	if err != nil {
		return err
	}
	src, err := prov.Provide(fname)
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

// Usage prints help text to the user.
func (cmd *ClearCommand) Usage() string {
	return ClearUsage
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

// Usage prints help text to the user.
func (cmd *StatusCommand) Usage() string {
	return StatusUsage
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

// Usage prints help text to the user.
func (cmd *LookupCommand) Usage() string {
	return LookupUsage
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
	fmt.Println(veh.FlexString("\n", "  "))
	return nil
}

// DisableCommand disables a vehicle so it won't appear in lookups.
type DisableCommand struct {
	Hash string `short:"h" long:"hash" description:"Hash of vehicle to disable"`
}

// Usage prints help text to the user.
func (cmd *DisableCommand) Usage() string {
	return DisableUsage
}

// Execute runs the disable command.
func (cmd *DisableCommand) Execute(opts []string) error {
	return store.Disable(cmd.Hash)
}

// EnableCommand enables a previously disabled vehicle so it will reappear in lookups.
type EnableCommand struct {
	Hash string `short:"h" long:"hash" description:"Hash of vehicle to enable"`
}

// Usage prints help text to the user.
func (cmd *EnableCommand) Usage() string {
	return EnableUsage
}

// Execute runs the enable command.
func (cmd *EnableCommand) Execute(opts []string) error {
	return store.Enable(cmd.Hash)
}

// QueryCommand represents a query/search against the vehicle store.
type QueryCommand struct {
	Limit    uint   `short:"l" long:"limit" description:"Limit on number of vehicles to return" default:"0"`
	Type     string `short:"t" long:"type" description:"Type of vehicle to filter by: Car|Bus|Van|Truck|Trailer|Unknown"`
	Brand    string `short:"b" long:"brand" description:"Brand name to filter by, case insensitive"`
	Model    string `short:"m" long:"model" description:"Model name to filter by, case insensitive"`
	FuelType string `short:"f" long:"fuel-type" description:"Fuel-type to filter by, case insensitive"`
}

// Usage prints help text to the user.
func (cmd *QueryCommand) Usage() string {
	return QueryUsage
}

// Execute performs a query against the vehicle store.
func (cmd *QueryCommand) Execute(opts []string) error {
	var out io.Writer = os.Stdout
	q := vehicle.Query{
		Limit:    int64(cmd.Limit),
		Type:     cmd.Type,
		Brand:    cmd.Brand,
		Model:    cmd.Model,
		FuelType: cmd.FuelType,
	}
	return store.QueryTo(out, q)
}

// loadConfig loads the TOML configuration file with the specified name.
func loadConfig(fname string) (config.Config, error) {
	return config.NewConfig(fname)
}

// bootstrap loads the app configuration and connects to the vehicle store.
func bootstrap(cmd flags.Commander, args []string) error {
	// Do not bootstrap the usual stuff when we are simply generating a config file.
	// @TODO: We're bypassing go-flags here. Consider refactoring this to make it less fragile.
	if len(os.Args) > 1 && os.Args[1] == "init" {
		return cmd.Execute(args)
	}
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

	// Initialise and configure the lookup manager.
	// @TODO: We need to loop over LookupConfigs and add a manager for each, dynamically.
	lookupManager = extlookup.NewManager()
	nrplade := extlookup.NewNrpladeService(conf.Providers["DMR"].LookupConfig)
	lookupManager.AddService(nrplade)

	// Carry on with command execution.
	return cmd.Execute(args)
}

// Start runs a command.
func Start() error {
	// This will run the command that matches the command-line options.
	if _, err := parser.Parse(); err != nil {

		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type == flags.ErrHelp {
			return nil // This is to avoid help messages from being printed twice.
		}
		return err
	}
	return nil
}
