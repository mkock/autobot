package app

import (
	"fmt"

	"github.com/mkock/autobot/config"
)

// init registers the command with the parser.
func init() {
	var initCmd InitCommand
	parser.AddCommand("init", "initialise", "write an empty configuration file that you can fill out", &initCmd)
}

// InitCommand writes an empty configuration file to cwd, to help the user getting started with autobot.
type InitCommand struct{}

// Usage prints help text to the user.
func (cmd *InitCommand) Usage() string {
	return InitUsage
}

// Execute writes the empty configuration file to the current working directory.
func (cmd *InitCommand) Execute(opts []string) error {
	if err := config.WriteEmptyConf("config.toml"); err != nil {
		return err
	}
	fmt.Println("Wrote config.toml")
	return nil
}
