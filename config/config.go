package config

import (
	"github.com/BurntSushi/toml"
)

// Config contains the application configuration.
type Config struct {
	Ftp FtpConfig
}

// FtpConfig contains FTP connection configuration.
type FtpConfig struct {
	Host       string
	Port       int
	User       string
	Password   string
	Dir        string
	FilePrefix string
}

// NewConfig returns a app configuration struct, loaded from a TOML file.
func NewConfig(fname string) (Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(fname, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}
