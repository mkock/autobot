package vehicle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/mkock/autobot/config"
)

// Exported errors.
var (
	ErrNoSuchVehicle = errors.New("no such vehicle")
)

// Store represents a Redis-compatible memory store such as Redis or Google Memory Store.
type Store struct {
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

// LogEntry contains two parts: a timestamp (logging time) and a message.
type LogEntry struct {
	LoggedAt time.Time
	Message  string
}

// String displays a human readable log message.
func (e LogEntry) String() string {
	return e.LoggedAt.Format("2006-01-02T15:04:04 ") + e.Message
}

// String returns a string with some status information on the operation.
func (op *syncOp) String() string {
	return fmt.Sprintf("%s sync status - began: %s, duration: %s. Summary: synced %d of %d vehicles", strings.ToUpper(op.source), op.started.Format("2006-01-02T15:04:05"), op.duration.Truncate(time.Second), op.synced, op.processed)
}

// NewStore returns a new Store, which you can then interact with in order to start sync operations etc.
func NewStore(storeCnf config.MemStoreConfig, syncCnf config.SyncConfig) *Store {
	return &Store{cnf: storeCnf, opts: syncCnf}
}

// Open connects to the vehicle store.
func (vs *Store) Open() error {
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
func (vs *Store) Close() error {
	return vs.store.Close()
}

// NewSyncOp starts a new synchronization operation and returns its id.
// Use this id for any further interactions with the operation. Note that his function is not thread-safe.
func (vs *Store) NewSyncOp(source string) SyncOpID {
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

func (vs *Store) getOp(id SyncOpID) *syncOp {
	// Find the referenced sync op.
	if int(id) > len(vs.ops)-1 {
		panic(fmt.Sprintf("no syncOp with id %d", id))
	}
	return &vs.ops[id]
}

// finalize calculates the total duration of the sync operation and pushes a stringified status result to mem-store.
func (vs *Store) finalize(id SyncOpID) {
	op := vs.getOp(id)
	end := time.Now()
	op.duration = end.Sub(op.started)
	if _, err := vs.store.LPush("ops", op.String()).Result(); err != nil {
		// @TODO: Perhaps we don't need to panic here?
		panic(fmt.Sprintf("unable to finalize sync operation"))
	}
}

// writeToFile is good to have around for debugging purposes.
func (vs *Store) writeToFile(vehicles List, outFile string) {
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
// @TODO Consider running "syncVehicle" in a go routine for faster execution speed.
func (vs *Store) Sync(id SyncOpID, vehicles <-chan Vehicle, done <-chan bool) error {
	op := vs.getOp(id)
	for {
		select {
		case vehicle := <-vehicles:
			op.processed++
			ok, err := vs.SyncVehicle(vehicle)
			if err != nil {
				return err
			}
			if ok {
				op.synced++
			}
		case <-done:
			vs.Log(op.String())
			vs.finalize(id)
			return nil
		}
	}
}

// serializeVehicle converts the given Vehicle to a string using JSON encoding.
func serializeVehicle(vehicle Vehicle) (string, error) {
	b, err := json.Marshal(vehicle)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unserializeVehicle converts the given string to a Vehicle using JSON decoding.
func unserializeVehicle(str string) (Vehicle, error) {
	var vehicle Vehicle
	if err := json.Unmarshal([]byte(str), &vehicle); err != nil {
		return vehicle, err
	}
	return vehicle, nil
}

// updateVehicle stores any changes made to the vehicle.
// Note: for now, this function assumes that changes were made only to the metadata. If the vehicle base data
// is changed, we need to generate a new hash and update the indexes.
func (vs *Store) updateVehicle(vehicle Vehicle) error {
	val, err := serializeVehicle(vehicle)
	if err != nil {
		return err
	}
	// Store the vehicle.
	hash := HashAsKey(vehicle.MetaData.Hash)
	if _, err = vs.store.HSet(vs.opts.VehicleMap, hash, val).Result(); err != nil {
		return err
	}
	return nil
}

// SyncVehicle synchronizes a single Vehicle with the memory store.
// It returns a bool indicating whether the vehicle was added/updated or not.
func (vs *Store) SyncVehicle(vehicle Vehicle) (bool, error) {
	mapName := vs.opts.VehicleMap
	vinIndex := vs.opts.VINSortedSet
	regIndex := vs.opts.RegNrSortedSet
	hash := HashAsKey(vehicle.MetaData.Hash)
	exists, err := vs.store.HExists(mapName, hash).Result()
	if err != nil || exists {
		return false, err
	}
	if err := vs.updateVehicle(vehicle); err != nil {
		return false, err
	}
	// Update the VIN index.
	zVIN := redis.Z{Score: 0, Member: fmt.Sprintf("%d:%s:%s", vehicle.MetaData.Country, vehicle.VIN, hash)}
	if _, err = vs.store.ZAdd(vinIndex, zVIN).Result(); err != nil {
		return false, err
	}
	// Update the reg.nr index.
	zReg := redis.Z{Score: 0, Member: fmt.Sprintf("%d:%s:%s", vehicle.MetaData.Country, vehicle.RegNr, hash)}
	if _, err = vs.store.ZAdd(regIndex, zReg).Result(); err != nil {
		return false, err
	}
	return true, nil
}

// Status returns a status for the sync operation with the given id.
func (vs *Store) Status(id SyncOpID) string {
	op := vs.getOp(id)
	return op.String()
}

// lookup attempts to lookup the given id (VIN or registration number) in the given index.
// The id type must match the index which is being used, otherwise there will never be a match.
// If a match was found, the vehicle hash is returned.
func (vs *Store) lookup(id, index string) (string, error) {
	zBy := redis.ZRangeBy{
		Min: fmt.Sprintf("[%s", id),
		Max: fmt.Sprintf("[%s\xff", id),
	}
	matches, err := vs.store.ZRangeByLex(index, zBy).Result()
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", nil // No match.
	}
	return strings.Split(matches[0], ":")[2], nil
}

// Enable enables the vehicle with the given hash value, if it exists.
func (vs *Store) Enable(hash string) error {
	vehicle, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return err
	}
	if vehicle == (Vehicle{}) {
		return ErrNoSuchVehicle
	}
	vehicle.MetaData.Disabled = false
	return vs.updateVehicle(vehicle)
}

// Disable disables the vehicle with the given hash value, if it exists.
func (vs *Store) Disable(hash string) error {
	vehicle, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return err
	}
	if vehicle == (Vehicle{}) {
		return ErrNoSuchVehicle
	}
	vehicle.MetaData.Disabled = true
	return vs.updateVehicle(vehicle)
}

// remove removes the member with the given id from the sorted set index of the given name.
func (vs *Store) remove(id, index string) error {
	if _, err := vs.store.ZRem(index, id).Result(); err != nil {
		return err
	}
	return nil
}

// LookupByVIN attempts to lookup a vehicle by its VIN number.
func (vs *Store) LookupByVIN(rc RegCountry, VIN string, showDisabled bool) (Vehicle, error) {
	val := strconv.Itoa(int(rc)) + ":" + strings.ToUpper(VIN)
	hash, err := vs.lookup(val, vs.opts.VINSortedSet)
	if err != nil || hash == "" {
		return Vehicle{}, err
	}
	return vs.lookupVehicle(hash, showDisabled, val, vs.opts.VINSortedSet)
}

// LookupByRegNr attempts to lookup a vehicle by its registration number.
func (vs *Store) LookupByRegNr(rc RegCountry, regNr string, showDisabled bool) (Vehicle, error) {
	val := strconv.Itoa(int(rc)) + ":" + strings.ToUpper(regNr)
	hash, err := vs.lookup(val, vs.opts.RegNrSortedSet)
	if err != nil || hash == "" {
		return Vehicle{}, err
	}
	return vs.lookupVehicle(hash, showDisabled, val, vs.opts.RegNrSortedSet)
}

// LookupByHash performs a vehicle lookup by hash value, without side effects. Ie. it doesn't attempt to clear
// any indexes if no vehicle was found.
func (vs *Store) LookupByHash(hash string) (Vehicle, error) {
	return vs.lookupVehicleSimple(hash)
}

// lookupVehicleSimple performs a vehicle lookup without any other processing.
func (vs *Store) lookupVehicleSimple(hash string) (Vehicle, error) {
	exists, err := vs.store.HExists(vs.opts.VehicleMap, hash).Result()
	if err != nil {
		return Vehicle{}, err
	}
	if exists {
		str, err := vs.store.HGet(vs.opts.VehicleMap, hash).Result()
		if err != nil {
			return Vehicle{}, err
		}
		return unserializeVehicle(str)
	}
	return Vehicle{}, err
}

// lookupVehicle attempts to locate the vehicle with the given hash in the vehicle store.
// If a vehicle was not found, it will attempt to delete the key from the index that was used for the lookup.
// The parameters "identifier" and "index" is the registration/VIN number and index name; they are only needed to
// reconstruct the index key that should be removed.
// Disabled vehicles will be treated as if they don't exist.
func (vs *Store) lookupVehicle(hash string, showDisabled bool, identifier, index string) (Vehicle, error) {
	vehicle, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return vehicle, err
	}
	if vehicle == (Vehicle{}) {
		// The index returned a hash value, but it does not exist in the vehicle store, so we delete the index.
		if err := vs.remove(fmt.Sprintf("%s:%s", identifier, hash), index); err != nil {
			fmt.Printf("Notice: unable to remove disconnected index for vehicle id %s", hash)
		}
	} else if vehicle.MetaData.Disabled && !showDisabled {
		return Vehicle{}, nil
	}
	return vehicle, nil
}

// Clear clears out the entire vehicle store, including indexes and the sync history.
func (vs *Store) Clear() error {
	keys := [...]string{vs.opts.SyncedFileString, vs.opts.VehicleMap, vs.opts.RegNrSortedSet, vs.opts.VINSortedSet, vs.opts.HistorySortedSet}
	if _, err := vs.store.Del(keys[:]...).Result(); err != nil {
		return err
	}
	return nil
}

// GetLastSynced returns the filename of the last file that was synchronised with the vehicle store.
// It returns an empty string if there is no filename.
func (vs *Store) GetLastSynced() (string, error) {
	return vs.store.Get(vs.opts.SyncedFileString).Result()
}

// SetLastSynced replaces the logged filename of the file that was last synchronised with the vehicle store.
func (vs *Store) SetLastSynced(fname string) error {
	if _, err := vs.store.Set(vs.opts.SyncedFileString, fname, 0).Result(); err != nil {
		return err
	}
	return nil
}

// Log logs a message to the vehicle store history together with the logging time.
func (vs *Store) Log(msg string) error {
	now := time.Now()
	zLog := redis.Z{Score: 0, Member: now.Format("20060102T150405") + ":" + msg}
	if _, err := vs.store.ZAdd(vs.opts.HistorySortedSet, zLog).Result(); err != nil {
		return err
	}
	return nil
}

// unmarshalLogEntry takes a string containing exactly one colon and returns a LogEntry with a timestamp parsed
// from the first value (before the colon), and an unparsed message (after the colon).
func unmarshalLogEntry(entry string) (LogEntry, error) {
	log := LogEntry{}
	if !strings.Contains(entry, ":") {
		return log, fmt.Errorf("history: log entry contains an unrecognised format: %s", entry)
	}
	parts := strings.SplitAfterN(entry, ":", 2)
	loggedAt, err := time.Parse("20060102T150405", parts[0][0:len(parts[0])-1])
	if err != nil {
		return log, err
	}
	log.LoggedAt = loggedAt
	log.Message = parts[1]
	return log, nil
}

// LastLog returns the message that was last logged in the history.
func (vs *Store) LastLog() (LogEntry, error) {
	logs, err := vs.store.ZRange(vs.opts.HistorySortedSet, -1, -1).Result()
	if err != nil {
		return LogEntry{}, err
	}
	return unmarshalLogEntry(logs[0])
}

// CountLog returns the number of log entries.
func (vs *Store) CountLog() (int, error) {
	count, err := vs.store.ZCount(vs.opts.HistorySortedSet, "0", "0").Result()
	return int(count), err
}
