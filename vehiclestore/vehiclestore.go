package vehiclestore

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/OmniCar/autobot/autoservice"
	"github.com/OmniCar/autobot/config"
	"github.com/go-redis/redis"
)

// VehicleStore represents a Redis-compatible memory store such as Redis or Google Memory Store.
type VehicleStore struct {
	cnf   config.MemStoreConfig
	opts  config.SyncConfig
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
func NewVehicleStore(storeCnf config.MemStoreConfig, syncCnf config.SyncConfig) *VehicleStore {
	return &VehicleStore{cnf: storeCnf, opts: syncCnf}
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

func (vs *VehicleStore) getOp(id SyncOpID) *syncOp {
	// Find the referenced sync op.
	if int(id) > len(vs.ops)-1 {
		panic(fmt.Sprintf("no syncOp with id %d", id))
	}
	return &vs.ops[id]
}

// finalize calculates the total duration of the sync operation and pushes a stringified status result to mem-store.
func (vs *VehicleStore) finalize(id SyncOpID) {
	op := vs.getOp(id)
	end := time.Now()
	op.duration = end.Sub(op.started)
	if _, err := vs.store.LPush("ops", op.String()).Result(); err != nil {
		// @TODO: Perhaps we don't need to panic here?
		panic(fmt.Sprintf("unable to finalize sync operation"))
	}
}

// writeToFile is good to have around for debugging purposes.
func (vs *VehicleStore) writeToFile(vehicles autoservice.VehicleList, outFile string) {
	out, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Unable to open output file %v for writing.\n", outFile)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	// Write CSV data.
	for _, vehicle := range vehicles {
		_, err := out.WriteString(vehicle.String() + "\n")
		if err != nil {
			fmt.Println("Unable to write to output file, unknown write error")
		}
	}
	fmt.Println("Wrote processed vehicle list to out.csv")
}

// Sync reads from channel "vehicles" and synchronizes each one with the store. It stops when receiving a bool on
// channel "done". Along the way, it keeps track of the number of vehicles that were processed and synchronized.
// This data is stored on the syncOp.
func (vs *VehicleStore) Sync(id SyncOpID, vehicles <-chan autoservice.Vehicle, done <-chan bool) error {
	op := vs.getOp(id)
	// @TODO Remove VehicleList and stream the vehicles directly to a file?
	var vlist autoservice.VehicleList = make(map[uint64]autoservice.Vehicle) // For keeping track of vehicles.
	for {
		select {
		case vehicle := <-vehicles:
			if _, ok := vlist[vehicle.MetaData.Ident]; !ok {
				vlist[vehicle.MetaData.Ident] = vehicle
			}
			op.processed++
			ok, err := vs.syncVehicle(vehicle)
			if err != nil {
				return err
			}
			if ok {
				op.synced++
			}
		case <-done:
			vs.writeToFile(vlist, "out.csv")
			vs.finalize(id)
			return nil
		}
	}
}

// serializeVehicle converts the given Vehicle to a string using JSON encoding.
func serializeVehicle(vehicle autoservice.Vehicle) (string, error) {
	b, err := json.Marshal(vehicle)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unserializeVehicle converts the given string to a Vehicle using JSON decoding.
func unserializeVehicle(str string) (autoservice.Vehicle, error) {
	var vehicle autoservice.Vehicle
	if err := json.Unmarshal([]byte(str), vehicle); err != nil {
		return vehicle, err
	}
	return vehicle, nil
}

// syncVehicle synchronizes a single Vehicle with the memory store.
// It returns a bool indicating whether the vehicle was added/updated or not.
func (vs *VehicleStore) syncVehicle(vehicle autoservice.Vehicle) (bool, error) {
	mapName := vs.opts.VehicleMap
	vinIndex := vs.opts.VINSortedSet
	regIndex := vs.opts.RegNrSortedSet
	hash := strconv.FormatUint(vehicle.MetaData.Hash, 10)
	exists, err := vs.store.HExists(mapName, hash).Result()
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	val, err := serializeVehicle(vehicle)
	if err != nil {
		return false, err
	}
	// Store the vehicle.
	if _, err = vs.store.HSet(mapName, hash, val).Result(); err != nil {
		return false, err
	}
	// Update the VIN index.
	zVIN := redis.Z{Score: 0, Member: fmt.Sprintf("%s:%s", vehicle.VIN, hash)}
	if _, err = vs.store.ZAdd(vinIndex, zVIN).Result(); err != nil {
		return false, err
	}
	// Update the reg.nr index.
	zReg := redis.Z{Score: 0, Member: fmt.Sprintf("%s:%s", vehicle.RegNr, hash)}
	if _, err = vs.store.ZAdd(regIndex, zReg).Result(); err != nil {
		return false, err
	}
	return true, nil
}

// Status returns a status for the sync operation with the given id.
func (vs *VehicleStore) Status(id SyncOpID) string {
	op := vs.getOp(id)
	return op.String()
}
