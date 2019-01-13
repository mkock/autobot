package app

import (
	"io"
	"os"

	"github.com/mkock/autobot/vehicle"
)

// init registers the command with the parser.
func init() {
	var queryCmd QueryCommand
	parser.AddCommand("query", "query vehicle store", "queries/searches the vehicle store using filters and selectors", &queryCmd)
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
