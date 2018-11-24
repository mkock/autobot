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
	config config.FtpConfig
	client *goftp.Client
}

// NewFtpProvider returns a new FtpProvider.
func NewFtpProvider(conf config.FtpConfig) *FtpProvider {
	return &FtpProvider{config: conf}
}

// Open establishes the FTP connection.
func (prov *FtpProvider) Open() error {
	dialConf := goftp.Config{
		User:               prov.config.User,
		Password:           prov.config.Password,
		ConnectionsPerHost: 1,
		Timeout:            12 * time.Hour,
	}
	var host string
	if prov.config.Port > 0 {
		host = fmt.Sprintf("%s:%d", prov.config.Host, prov.config.Port)
	} else {
		host = prov.config.Host
	}
	log.Printf("Connecting to %s...\n", host)
	client, dialErr := goftp.DialConfig(dialConf, host)
	if dialErr != nil {
		return dialErr
	}
	// Run a MLSD/LIST op to verify the connection, due to a flaw in goftp where it doesn't return an error when the
	// connection fails. See https://github.com/secsy/goftp/issues/35.
	if _, err := client.ReadDir(prov.config.Dir); err != nil {
		return err
	}
	prov.client = client
	return nil
}

// Close closes the FTP connection.
func (prov *FtpProvider) Close() error {
	return prov.client.Close()
}

// CheckForLatest checks if there are any new files in the same format as the one given and returns
// the filename of the latest one if possible. Otherwise, the original filename is returned.
func (prov *FtpProvider) CheckForLatest(fname string) (string, error) {
	files, err := prov.client.ReadDir(prov.config.Dir)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no such file %s", fname)
	}
	newest := fname
	if newest == "" {
		// Date/time is far in the past so comparisons will always benefit actual files.
		newest = fmt.Sprintf("%s20000101-000000.zip", prov.config.FilePrefix)
	}
	for _, file := range files {
		if isNewer(file.Name(), newest) {
			newest = file.Name()
		}
	}
	return newest, nil
}

// Provide make an FTP file available to autobot by downloading it.
func (prov *FtpProvider) Provide(fname string) (io.ReadCloser, error) {
	srcPath := filepath.Join(prov.config.Dir, fname)
	if _, statErr := prov.client.Stat(srcPath); statErr != nil {
		return nil, statErr
	}
	tmp := fmt.Sprintf("/tmp/%s", fname)
	w, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	log.Printf("Downloading %s...\n", fname)
	if err := prov.client.Retrieve(srcPath, w); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		return nil, err
	}
	r, err := os.Open(tmp)
	if isZipped(fname) {
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
