package args

import (
	"github.com/jessevdk/go-flags"
)

// CLIOptions contains command-line arguments parsed upon application initialisation.
type CLIOptions struct {
	ConfigFile string `short:"c" long:"config-file" required:"yes" default:"config.toml" description:"Application configuration file in TOML format"`
	SourceFile string `short:"f" long:"source-file" description:"DMR XML file in UTF-8 format"`
	VIN        string `short:"v" long:"vin" description:"VIN number to lookup, if any (will not synchronize data)"`
	RegNr      string `short:"r" long:"regnr" description:"Registration number to lookup, if any (will not synchronize data)"`
	Debug      bool   `short:"d" long:"debug" description:"Debug: print CPU count, goroutine count and memory usage every 10 seconds"`
	Clear      bool   `short:"l" long:"clear" description:"Clears the entire vehicle store"`
	Status     bool   `short:"s" long:"status" description:"Displays the last synchronisation log"`
}

// ParseCLI wraps the go-flags parser to keep the dependency out of your hair.
func ParseCLI() (CLIOptions, error) {
	var opts CLIOptions
	_, err := flags.Parse(&opts)
	return opts, err
}
