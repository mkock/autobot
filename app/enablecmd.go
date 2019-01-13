package app

// init registers the command with the parser.
func init() {
	var enableCmd EnableCommand
	parser.AddCommand("enable", "enable vehicle", "enables a vehicle so it will reappear in lookups", &enableCmd)
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
