package app

import (
	"fmt"
	"log"

	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/dataprovider"
	"github.com/mkock/autobot/dmr"
)

// init registers the command with the parser.
func init() {
	var syncCmd SyncCommand
	parser.AddCommand("sync", "synchronise", "synchronise the vehicle store with an external data source", &syncCmd)
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

// IsConnected reports whether or not this command needs to connect to the VehicleStore.
func (cmd *SyncCommand) IsConnected() bool {
	return true
}
