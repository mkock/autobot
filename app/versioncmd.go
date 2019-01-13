package app

import "fmt"

// init registers the command with the parser.
func init() {
	var verCmd VersionCommand
	parser.AddCommand("version", "version", "displays the current build version of autobot", &verCmd)
}

// VersionCommand displays the current build version of autobot.
type VersionCommand struct{}

// Usage prints help text to the user.
func (cmd *VersionCommand) Usage() string {
	return VersionUsage
}

// Execute displays the current build version of autobot.
func (cmd *VersionCommand) Execute(opts []string) error {
	fmt.Printf("Autobot version: %s\n", version)
	return nil
}

// IsConnected reports whether or not this command needs to connect to the VehicleStore.
func (cmd *VersionCommand) IsConnected() bool {
	return false
}
