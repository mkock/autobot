package vehiclestore

import (
	"fmt"
	"time"

	"github.com/OmniCar/autobot/autoservice"
	"github.com/OmniCar/autobot/config"
	"github.com/go-redis/redis"
)

// VehicleStore represents a Redis-compatible memory store such as Redis or Google Memory Store.
type VehicleStore struct {
	cnf   config.MemStoreConfig
	store *redis.Client
	ops   []syncOp
}

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
	return fmt.Sprintf("Sync from %s started: %s, duration: %s. Summary: synced %d of %d vehicles", op.source, op.started.Format("2006-01-02T15:04:05"), op.duration, op.synced, op.processed)
}

// NewVehicleStore returns a new VehicleStore, which you can then interact with in order to start sync operations etc.
func NewVehicleStore(cnf config.MemStoreConfig) *VehicleStore {
	return &VehicleStore{cnf: cnf}
}

// Open connects to the vehicle store.
func (vs *VehicleStore) Open() error {
	vs.store = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", vs.cnf.Host, vs.cnf.Port),
		Password: vs.cnf.Password,
		DB:       vs.cnf.DB,
	})
	if _, err := vs.store.Ping().Result(); err != nil {
		return err
	}
	return nil
}

// Close disconnects from the memory store.
func (vs *VehicleStore) Close() error {
	return vs.store.Close()
}

// NewSyncOp starts a new synchronization operation and returns its id.
// Use this id for any further interactions with the operation. Note that his function is not thread-safe.
func (vs *VehicleStore) NewSyncOp(source string) SyncOpID {
	id := SyncOpID(len(vs.ops))
	op := syncOp{
		id:        id,
		started:   time.Now(),
		source:    source,
		processed: 0,
		synced:    0,
	}
	vs.ops = append(vs.ops, op)
	return id
}

// Sync reads from channel "vehicles" and synchronizes each one with the store. It stops when receiving a bool on
// channel "done".
func (vs *VehicleStore) Sync(id SyncOpID, vehicles <-chan autoservice.Vehicle, done <-chan bool) {
	if int(id) > len(vs.ops)-1 {
		panic(fmt.Sprintf("Autobot: no syncOp with id %d", id))
	}
	op := vs.ops[id]

	// Finalize sync operation.
	end := time.Now()
	op.duration = end.Sub(op.started)
	if _, err := vs.store.LPush("ops", op.String()).Result(); err != nil {
		panic(fmt.Sprintf("Autobot: unable to finalize sync operation"))
	}
}
