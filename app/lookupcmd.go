package app

import (
	"fmt"

	"github.com/mkock/autobot/vehicle"
)

// init registers the command with the parser.
func init() {
	var lookupCmd LookupCommand
	parser.AddCommand("lookup", "lookup vehicle", "performs a vehicle lookup, by VIN or registration number", &lookupCmd)
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
