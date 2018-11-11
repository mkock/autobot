package dataprovider

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/OmniCar/autobot/config"
)

// Constants for picking DataProvider implementations.
const (
	FtpProv = iota
	FsProv
)

// DataProvider is the interface for implementations that fetches files for Autobot to parse.
type DataProvider interface {
	Open() error
	Close() error
	CheckForLatest(string) (string, error)
	Provide(string) (io.ReadCloser, error)
}

// ProvTypeString returns the string representation of the provider type.
func ProvTypeString(ptype int) string {
	switch ptype {
	case FtpProv:
		return "ftp"
	case FsProv:
		return "fs"
	default:
		return ""
	}
}

// NewProvider returns a new provider of the requested type (implementation).
func NewProvider(ptype int, config config.Config) DataProvider {
	switch ptype {
	case FtpProv:
		return NewFtpProvider(config.Ftp)
	case FsProv:
		return NewFileProvider()
	default:
		log.Fatalf("No such provider: %d (%s)", ptype, ProvTypeString(ptype))
	}
	return nil
}

// isZipped checks if the given file name has the ".zip" extension.
func isZipped(fname string) bool {
	return filepath.Ext(fname) == ".zip"
}

// unzip extracts the source ReadCloser by writing to a zip file, and then extracting it into a new io.ReadCloser.
func unzip(src io.ReadCloser) (io.ReadCloser, error) {
	// Write the zip file to a temporary file on disk.
	tmp := "/tmp/vehicledata.zip"
	file, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	defer func() {
		file.Close()
		src.Close()
	}()
	if _, err := io.Copy(file, src); err != nil {
		return nil, err
	}
	// Open the temporary file and extract the first zipped file.
	r, err := zip.OpenReader(tmp)
	if err != nil {
		return nil, err
	}
	if len(r.File) == 0 {
		return nil, fmt.Errorf("unzip %s: empty zip file", tmp)
	}
	f := r.File[0] // Assuming that the zip file contains only a single file.
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	return rc, nil
}
