package vehicle

import (
	"fmt"
	"strings"
	"time"
)

// LogEntry contains two parts: a timestamp (logging time) and a message.
type LogEntry struct {
	LoggedAt time.Time
	Message  string
}

// String displays a human readable log message.
func (log LogEntry) String() string {
	return log.LoggedAt.Format("2006-01-02T15:04:04 ") + log.Message
}

// Unmarshal takes a string containing exactly one colon and populates the LogEntry with a timestamp parsed
// from the first value (before the colon), and an unparsed message (after the colon).
func (log *LogEntry) Unmarshal(str string) error {
	if !strings.Contains(str, ":") {
		return fmt.Errorf("history: log entry contains an unrecognised format: %s", str)
	}
	parts := strings.SplitAfterN(str, ":", 2)
	var err error
	if log.LoggedAt, err = time.Parse("20060102T150405", parts[0][0:len(parts[0])-1]); err != nil {
		return err
	}
	log.Message = parts[1]
	return nil
}
