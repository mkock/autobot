package app

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/extlookup"
	"github.com/mkock/autobot/vehicle"
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
// Each command adds themselves via their own init functions.
func init() {
	// Setup commands.
	globalOpts = &Options{}
	parser = flags.NewParser(globalOpts, flags.Default)
	parser.CommandHandler = bootstrap // Loads conf and connects to the vehicle store before executing commands.
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

// connecter is an interface that, if satisfied by a go-flags Commander, allows us to skip connecting to the
// VehicleStore for commands where "isConnected" returns false.
type connecter interface {
	IsConnected() bool
}

// Options contains command-line arguments parsed upon application initialisation.
type Options struct {
	ConfigFile string `short:"c" long:"config-file" required:"no" default:"config.toml" description:"Application configuration file in TOML format"`
}

// loadConfig loads the TOML configuration file with the specified name.
func loadConfig(fname string) (config.Config, error) {
	return config.NewConfig(fname)
}

// bootstrap loads the app configuration and connects to the vehicle store.
func bootstrap(cmd flags.Commander, args []string) error {
	// Do not bootstrap the usual stuff when we are not dealing with a connected command. Instead, return early.
	if t, ok := cmd.(connecter); ok && !t.IsConnected() {
		return cmd.Execute(args)
	}
	// Load the configuration file passed via the CLI.
	var err error
	conf, err = loadConfig(globalOpts.ConfigFile)
	if err != nil {
		return err
	}

	// Connect to the vehicle store.
	store = vehicle.NewStore(conf.MemStore, conf.Sync, os.Stdout)
	if err := store.Open(); err != nil {
		return err
	}
	defer store.Close()

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
