package dataprovider

import (
	"fmt"
	"io"
	"os"
)

// FileProvider is a data provider that supports local files.
type FileProvider struct {
	fname string
}

// NewFileProvider returns a new FileProvider.
func NewFileProvider() *FileProvider {
	return &FileProvider{}
}

// Open checks if the file is a readable file.
func (prov *FileProvider) Open(fname string) error {
	finfo, err := os.Stat(fname)
	if err != nil {
		return err
	}
	if finfo.IsDir() {
		return fmt.Errorf("filename %s is a directory", fname)
	}
	prov.fname = fname
	return nil
}

// Close does nothing.
func (prov *FileProvider) Close() error {
	return nil
}

// CheckForLatest does nothing for FileProvider.
func (prov *FileProvider) CheckForLatest() (string, error) {
	return prov.fname, nil
}

// Provide makes a local file available to autobot.
func (prov *FileProvider) Provide() (rc io.ReadCloser, err error) {
	rc, err = os.Open(prov.fname)
	if err != nil {
		return nil, err
	}
	if isZipped(prov.fname) {
		return unzip(rc)
	}
	return rc, nil
}
