package dataprovider

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DataProvider is the interface for implementations that fetches files for Autobot to parse.
type DataProvider interface {
	Open(fname string) error
	Close() error
	CheckForLatest() (string, error)
	Provide() (*io.ReadCloser, error)
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
