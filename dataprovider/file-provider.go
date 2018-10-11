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

// CheckForLatest does nothing for FileProvider.
func (prov *FileProvider) CheckForLatest(fname string) (string, error) {
	return fname, nil
}

// Provide makes a local file available to autobot.
func (prov *FileProvider) Provide(fname string) (rc io.ReadCloser, err error) {
	finfo, err := os.Stat(fname)
	if err != nil {
		return nil, err
	}
	if finfo.IsDir() {
		return nil, fmt.Errorf("Autobot: filename %s is a directory", fname)
	}
	if isZipped(fname) {
		return unzip(fname)
	}
	return os.Open(fname)
}
