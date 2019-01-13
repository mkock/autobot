package app

// init registers the command with the parser.
func init() {
	var disableCmd DisableCommand
	parser.AddCommand("disable", "disable vehicle", "disables a vehicle so it won't appear in lookups", &disableCmd)
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

// IsConnected reports whether or not this command needs to connect to the VehicleStore.
func (cmd *DisableCommand) IsConnected() bool {
	return true
}
