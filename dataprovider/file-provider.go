package dataprovider

import (
	"fmt"
	"io"
	"os"
)

// FileProvider is a data provider that supports local files.
type FileProvider struct{}

// NewFileProvider returns a new FileProvider.
func NewFileProvider() *FileProvider {
	return &FileProvider{}
}

// Open does nothing for FileProvider.
func (prov *FileProvider) Open() error {
	return nil
}

// Close does nothing.
func (prov *FileProvider) Close() error {
	return nil
}

// CheckForLatest simply checks if the file is readable.
func (prov *FileProvider) CheckForLatest(fname string) (string, error) {
	finfo, err := os.Stat(fname)
	if err != nil {
		return "", err
	}
	if finfo.IsDir() {
		return "", fmt.Errorf("filename %s is a directory", fname)
	}
	return fname, nil
}

// Provide makes a local file available to autobot.
func (prov *FileProvider) Provide(fname string) (rc io.ReadCloser, err error) {
	rc, err = os.Open(fname)
	if err != nil {
		return nil, err
	}
	if isZipped(fname) {
		return unzip(rc)
	}
	return rc, nil
}
