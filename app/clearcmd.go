package app

// init registers the command with the parser.
func init() {
	var clearCmd ClearCommand
	parser.AddCommand("clear", "clear", "clears the vehicle store of all vehicles", &clearCmd)
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
