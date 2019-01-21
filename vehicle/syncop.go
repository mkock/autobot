package vehicle

import (
	"fmt"
	"strings"
	"time"
)

// SyncOpID is an integer reference to a running synchronization operation.
type SyncOpID int

// syncOp represents a synchronization operation: when it started, how long it took, where it synced from
// and how many vehicles were processed and synced, respectively.
type syncOp struct {
	id        SyncOpID
	started   time.Time
	duration  time.Duration
	source    string
	processed int
	synced    int
}

// String returns a string with some status information on the operation.
func (op *syncOp) String() string {
	return fmt.Sprintf("%s sync status - began: %s, duration: %s. Summary: synced %d of %d vehicles", strings.ToUpper(op.source), op.started.Format("2006-01-02T15:04:05"), op.duration.Truncate(time.Second), op.synced, op.processed)
}

// End sets the end time of the operation and calculates the duration.
func (op *syncOp) End() {
	end := time.Now()
	op.duration = end.Sub(op.started)
}
