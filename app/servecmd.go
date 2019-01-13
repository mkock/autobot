package app

import (
	"log"

	"github.com/mkock/autobot/webservice"
)

// init registers the command with the parser.
func init() {
	var serveCmd ServeCommand
	parser.AddCommand("serve", "serve", "starts autobot as a web server", &serveCmd)
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

// IsConnected reports whether or not this command needs to connect to the VehicleStore.
func (cmd *ServeCommand) IsConnected() bool {
	return true
}
