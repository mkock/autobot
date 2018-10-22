package dataprovider

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/OmniCar/autobot/config"
	"github.com/secsy/goftp"
)

// FtpProvider is a data provider that supports file retrieval via FTP.
type FtpProvider struct {
	fname  string
	config config.FtpConfig
	client *goftp.Client
}

// NewFtpProvider returns a new FtpProvider.
func NewFtpProvider(conf config.FtpConfig) *FtpProvider {
	return &FtpProvider{config: conf}
}

// Open establishes the FTP connection.
func (prov *FtpProvider) Open(fname string) error {
	prov.fname = fname
	dialConf := goftp.Config{
		User:               prov.config.User,
		Password:           prov.config.Password,
		ConnectionsPerHost: 1,
		Timeout:            12 * time.Hour,
	}
	fmt.Printf("Connecting to %s:%d...\n", prov.config.Host, prov.config.Port)
	client, dialErr := goftp.DialConfig(dialConf, prov.config.Host+":"+strconv.Itoa(prov.config.Port))
	if dialErr != nil {
		return dialErr
	}
	// Run a STAT op to verify the connection, due to a bug in goftp where it doesn't return an error when the
	// connection fails.
	_, err := client.Stat("/")
	if err != nil {
		return err
	}
	prov.client = client
	return nil
}

// Close closes the FTP connection.
func (prov *FtpProvider) Close() error {
	return prov.client.Close()
}

// CheckForLatest checks if there are any new files in the same format as the one given and returns the filename
// of the latest one if possible. Otherwise, the original filename is assigned.
func (prov *FtpProvider) CheckForLatest() (string, error) {
	files, err := prov.client.ReadDir(prov.config.Dir)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no such file %s", prov.fname)
	}
	newest := prov.fname
	if newest == "" {
		// Date/time is in the past so comparisons will always benefit actual files.
		newest = fmt.Sprintf("%s20000101-000000.zip", prov.config.FilePrefix)
	}
	for _, file := range files {
		if isNewer(file.Name(), newest) {
			newest = file.Name()
		}
	}
	prov.fname = newest
	return newest, nil
}

// Provide make an FTP file available to autobot by downloading it.
func (prov *FtpProvider) Provide() (io.ReadCloser, error) {
	srcPath := filepath.Join(prov.config.Dir, prov.fname)
	_, statErr := prov.client.Stat(srcPath)
	if statErr != nil {
		return nil, statErr
	}
	tmp := fmt.Sprintf("/tmp/%s", prov.fname)
	w, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	if err := prov.client.Retrieve(srcPath, w); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		return nil, err
	}
	r, err := os.Open(tmp)
	if isZipped(prov.fname) {
		return unzip(r)
	}
	return r, nil
}

// isNewer tests whether the date/time part of file1 is newer than the date/time part of file2.
// Expected file format: ESStatistikListeModtag-YYYYMMDD-HHMMSS.zip.
func isNewer(file1, file2 string) bool {
	if file1 == file2 {
		return false
	}
	parts1 := strings.Split(strings.TrimSuffix(file1, ".zip"), "-")
	parts2 := strings.Split(strings.TrimSuffix(file2, ".zip"), "-")
	if len(parts1) != 3 || len(parts2) != 3 {
		return false // Uncomparable.
	}
	if parts1[1] == parts2[1] {
		// If date parts are identical, then compare time parts.
		time1, _ := strconv.Atoi(parts1[2])
		time2, _ := strconv.Atoi(parts2[2])
		return time1 > time2
	}
	// At this point, the date parts are not identical, so we compare them directly.
	date1, _ := strconv.Atoi(parts1[1])
	date2, _ := strconv.Atoi(parts2[1])
	return date1 > date2
}
