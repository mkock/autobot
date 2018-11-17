package config

import (
	"os"
	"text/template"
)

var cnfTpl = `[Ftp]
Host = ""
Port = 21
User = ""
Password = ""
Dir = "/"
FilePrefix = "ESStatistikListeModtag-"

[MemStore]
Host = "127.0.0.1"
Port = 6379
Password = ""
DB = 0

[WebService]
# Schedule follows the cron five-field syntax: "minute hours day-of-month month day-of-week".
# Example: "0 10 * * MON" runs the sync job every Monday at 10AM.
Schedule = "0 10 * * MON"

[Sync]
SyncedFileString = "autobot_synced"
VehicleMap = "autobot_vehicles"
VINSortedSet = "autobot_vin_index"
RegNrSortedSet = "autobot_regnr_index"
HistorySortedSet = "autobot_history"
`

// WriteEmptyConf writes a new, empty configuration file to the filename with the given name.
// The configuration file is in TOML format and contains some sensible defaults for most non-sensitive key/value pairs.
func WriteEmptyConf(fname string) error {
	tpl, err := template.New("config").Parse(cnfTpl)
	if err != nil {
		return err
	}
	fout, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer fout.Close()
	return tpl.Execute(fout, struct{}{})
}
