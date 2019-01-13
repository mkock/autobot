package app

import "fmt"

// init registers the command with the parser.
func init() {
	var statusCmd StatusCommand
	parser.AddCommand("status", "status", "displays a short status of the vehicle store", &statusCmd)
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
