package autobot

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const logFile = "status.log"

// LogAsProcessed will log the given filename as the last processed file from DMR.
func LogAsProcessed(fname string) error {
	file, error := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if error != nil {
		return error
	}
	defer func() {
		if error := file.Close(); error != nil {
			panic(error)
		}
	}()
	_, writeErr := file.WriteString(fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02T15:04"), fname))
	if writeErr != nil {
		return error
	}
	return nil
}

// GetLastProcessed returns the name of the last processed DMR file.
// This will be an empty string if nothing was logged.
func GetLastProcessed() (string, error) {
	file, error := os.Open(logFile)
	if error != nil {
		return "", error
	}
	defer func() {
		if error := file.Close(); error != nil {
			panic(error)
		}
	}()
	var line string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = scanner.Text()
	}
	parts := strings.Split(line, " ")
	if len(parts) > 1 {
		return parts[1], nil
	}
	return "", nil
}
