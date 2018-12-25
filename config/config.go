package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
)

// Config contains the application configuration.
type Config struct {
	Providers  map[string]ProviderConfig
	MemStore   MemStoreConfig
	WebService WebServiceConfig
	Sync       SyncConfig
}

// ProviderConfig contains configuration for the data provider.
type ProviderConfig struct {
	FtpConfig
	LookupConfig
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

// LookupConfig contains configuration for performing direct vehicle lookups via an API.
type LookupConfig struct {
	LookupSupported bool
	LookupSecure    bool
	LookupHost      string
	LookupPath      string
	LookupKey       string
}

// MemStoreConfig contains configuration for memory store / Redis.
type MemStoreConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// WebServiceConfig contains configuration related to the web service and sync scheduler.
type WebServiceConfig struct {
	Schedule string
}

type date struct {
	time.Time
}

// UnmarshalText is used by the TOML package to parse dates.
func (d *date) UnmarshalText(text []byte) error {
	var (
		str string
		err error
	)
	str = string(text)
	if str == "" {
		return nil
	}
	if d.Time, err = time.Parse("2006-01-02", str); err != nil {
		return err
	}
	return nil
}

// SyncConfig contains configuration related to the actual synchronization algorithm.
type SyncConfig struct {
	SyncedFileString string
	VehicleMap       string
	VINSortedSet     string
	RegNrSortedSet   string
	HistorySortedSet string
	EarliestRegDate  date
}

// NewConfig returns a app configuration struct, loaded from a TOML file.
// If a file path is included in the file name, the file will be loaded from that path. Otherwise, NewConfig will assume
// that the file is available in the same directory as the autobot executable and attempt to load it from there.
func NewConfig(fname string) (Config, error) {
	var conf Config
	file := findConfig(fname)
	if file == "" {
		return conf, fmt.Errorf("No such file: %s", fname)
	}
	if _, err := toml.DecodeFile(fname, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

// findConfig checks for the given file name in several locations: 1) the path, if a path is part of the file name,
// 2) the current working directory, and 3) the directory of the autobot executable. Returns an empty string if the
// file could not be found, or if it's a directory.
func findConfig(fname string) string {
	fpart := path.Base(fname)
	// If we have a path or the file exists in the current working directory, use it.
	if isRegularFile(fname) {
		return fname
	}
	// Check the directory of the autobot executable.
	dir, err := os.Executable()
	if err != nil {
		return ""
	}
	fname = path.Join(dir, fpart)
	if isRegularFile(fname) {
		return fname
	}
	return ""
}

func isRegularFile(fname string) bool {
	if fname == "" {
		return false
	}
	finfo, err := os.Stat(fname)
	if err != nil {
		return false
	}
	return !finfo.IsDir()
}
