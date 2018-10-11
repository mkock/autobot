package dataprovider

import (
	"archive/zip"
	"io"
	"path/filepath"
)

// DataProvider is the interface for implementations that fetches files for Autobot to parse.
type DataProvider interface {
	CheckForLatest(fname string) (string, error)
	Provide(fname string) (*io.ReadCloser, error)
}

// isZipped checks if the given file name has the ".zip" extension.
func isZipped(fname string) bool {
	return filepath.Ext(fname) == ".zip"
}

// unzip extracts the src file into the dest file.
func unzip(src string) (io.ReadCloser, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	f := r.File[0] // Assuming that the zip file contains a single file.
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	return rc, nil
}
