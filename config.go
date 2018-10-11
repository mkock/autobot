package autobot

import (
	"github.com/BurntSushi/toml"
	"github.com/OmniCar/autobot/dataprovider"
)

// Config contains the application configuration.
type Config struct {
	Ftp dataprovider.FtpConfig
}

// NewConfig returns a app configuration struct, loaded from a TOML file.
func NewConfig(fname string) (*Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(fname, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
