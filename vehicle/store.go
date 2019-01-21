package vehicle

import (
	"errors"
	"fmt"
	"io"
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
	cnf    config.MemStoreConfig
	opts   config.SyncConfig
	store  *redis.Client
	ops    []syncOp
	logger io.Writer
}

// NewStore returns a new Store, which you can then interact with in order to start sync operations etc.
func NewStore(storeCnf config.MemStoreConfig, syncCnf config.SyncConfig, logger io.Writer) *Store {
	return &Store{cnf: storeCnf, opts: syncCnf, logger: logger}
}

// Open connects to the vehicle store.
func (vs *Store) Open() error {
	vs.store = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", vs.cnf.Host, vs.cnf.Port),
		Password:     vs.cnf.Password,
		DB:           vs.cnf.DB,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
	_, err := vs.store.Ping().Result()
	return err
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
	op.End()
	if _, err := vs.store.LPush("ops", op.String()).Result(); err != nil {
		// @TODO: Perhaps we don't need to panic here?
		panic("unable to finalize sync operation")
	}
}

// writeToFile is good to have around for debugging purposes.
func (vs *Store) writeToFile(vehicles List, outFile string) {
	out, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(vs.logger, "Unable to open output file %v for writing.\n", outFile)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	// Write CSV data.
	for _, veh := range vehicles {
		_, err := out.WriteString(veh.String() + "\n")
		if err != nil {
			fmt.Fprintln(vs.logger, "Unable to write to output file, unknown write error")
		}
	}
	fmt.Fprintln(vs.logger, "Wrote processed vehicle list to out.csv")
}

// Sync reads from channel "vehicles" and synchronizes each one with the store. It stops when receiving a bool on
// channel "done". Along the way, it keeps track of the number of vehicles that were processed and synchronized.
// This data is stored on the syncOp.
func (vs *Store) Sync(id SyncOpID, vehicles <-chan Vehicle, done <-chan bool) error {
	var (
		ok      bool
		err     error
		vehicle Vehicle
	)
	op := vs.getOp(id)
	for {
		select {
		case vehicle = <-vehicles:
			op.processed++
			// Only synchronise vehicles that satisfy the limit on reg.date.
			if vehicle.FirstRegDate.After(vs.opts.EarliestRegDate.Time) {
				ok, err = vs.SyncVehicle(vehicle)
				if err != nil {
					return err
				}
				if ok {
					op.synced++
				}
			}
		case <-done:
			vs.Log(op.String())
			vs.finalize(id)
			return nil
		}
	}
}

// updateVehicle stores any changes made to the vehicle.
// Note: for now, this function assumes that changes were made only to the metadata. If the vehicle base data
// is changed, we need to generate a new hash and update the indexes.
func (vs *Store) updateVehicle(veh Vehicle) error {
	val, err := veh.Marshal()
	if err != nil {
		return err
	}
	// Store the vehicle.
	hash := HashAsKey(veh.MetaData.Hash)
	if _, err = vs.store.HSet(vs.opts.VehicleMap, hash, val).Result(); err != nil {
		return err
	}
	return nil
}

// SyncVehicle synchronizes a single Vehicle with the memory store.
// It returns a bool indicating whether the vehicle was added/updated or not.
func (vs *Store) SyncVehicle(veh Vehicle) (bool, error) {
	mapName := vs.opts.VehicleMap
	vinIndex := vs.opts.VINSortedSet
	regIndex := vs.opts.RegNrSortedSet
	hash := HashAsKey(veh.MetaData.Hash)
	exists, err := vs.store.HExists(mapName, hash).Result()
	if err != nil || exists {
		return false, err
	}
	if err := vs.updateVehicle(veh); err != nil {
		return false, err
	}
	// Update the VIN index.
	zVIN := redis.Z{Score: 0, Member: fmt.Sprintf("%d:%s:%s", veh.MetaData.Country, veh.VIN, hash)}
	if _, err = vs.store.ZAdd(vinIndex, zVIN).Result(); err != nil {
		return false, err
	}
	// Update the reg.nr index.
	zReg := redis.Z{Score: 0, Member: fmt.Sprintf("%d:%s:%s", veh.MetaData.Country, veh.RegNr, hash)}
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
	veh, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return err
	}
	if veh == (Vehicle{}) {
		return ErrNoSuchVehicle
	}
	veh.MetaData.Disabled = false
	return vs.updateVehicle(veh)
}

// Disable disables the vehicle with the given hash value, if it exists.
func (vs *Store) Disable(hash string) error {
	veh, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return err
	}
	if veh == (Vehicle{}) {
		return ErrNoSuchVehicle
	}
	veh.MetaData.Disabled = true
	return vs.updateVehicle(veh)
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
	var v Vehicle
	exists, err := vs.store.HExists(vs.opts.VehicleMap, hash).Result()
	if err != nil || !exists {
		return v, err
	}
	str, err := vs.store.HGet(vs.opts.VehicleMap, hash).Result()
	if err != nil {
		return v, err
	}
	v.Unmarshal(str)
	return v, nil
}

// lookupVehicle attempts to locate the vehicle with the given hash in the vehicle store.
// If a vehicle was not found, it will attempt to delete the key from the index that was used for the lookup.
// The parameters "identifier" and "index" is the registration/VIN number and index name; they are only needed to
// reconstruct the index key that should be removed.
// Disabled vehicles will be treated as if they don't exist.
func (vs *Store) lookupVehicle(hash string, showDisabled bool, identifier, index string) (Vehicle, error) {
	veh, err := vs.lookupVehicleSimple(hash)
	if err != nil {
		return veh, err
	}
	if veh == (Vehicle{}) {
		// The index returned a hash value, but it does not exist in the vehicle store, so we delete the index.
		if err := vs.remove(fmt.Sprintf("%s:%s", identifier, hash), index); err != nil {
			fmt.Fprintf(vs.logger, "Notice: unable to remove disconnected index for vehicle id %s", hash)
		}
	} else if veh.MetaData.Disabled && !showDisabled {
		return Vehicle{}, nil
	}
	return veh, nil
}

// Clear clears out the entire vehicle store, including indexes but not the sync history.
func (vs *Store) Clear() error {
	keys := [...]string{vs.opts.SyncedFileString, vs.opts.VehicleMap, vs.opts.RegNrSortedSet, vs.opts.VINSortedSet}
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

// LastLog returns the message that was last logged in the history.
func (vs *Store) LastLog() (LogEntry, error) {
	var (
		err  error
		log  LogEntry
		logs []string
	)
	if logs, err = vs.store.ZRange(vs.opts.HistorySortedSet, -1, -1).Result(); err != nil {
		return log, err
	}
	err = log.Unmarshal(logs[0])
	return log, err
}

// CountLog returns the number of log entries.
func (vs *Store) CountLog() (int, error) {
	count, err := vs.store.ZCount(vs.opts.HistorySortedSet, "0", "0").Result()
	return int(count), err
}

// QueryTo performs a query/search agsinst the store and streams the results to the provided reader.
func (vs *Store) QueryTo(w io.Writer, q Query) error {
	titles := []interface{}{"hash", "country", "ident", "reg nr", "vin", "brand", "model", "fuel type", "first reg date"}
	fmt.Fprintf(w, "%q,%q,%q,%q,%q,%q,%q,%q,%q\n", titles...)
	var (
		batch, progress int64
		keys            []string
		cur             uint64
		err             error
		res, props      []interface{}
		strVeh          string
		ok              bool
		veh             Vehicle
	)
	props = make([]interface{}, 10)
	batch = 100
	progress = 0
	// Prepare some querying parameters.
	pq := prepareQuery(q)
	// Loop over cursors 100 entries at a time.
	for {
		if keys, cur, err = vs.store.HScan(vs.opts.VehicleMap, cur, "", batch).Result(); err != nil {
			return err
		}
		if res, err = vs.store.HMGet(vs.opts.VehicleMap, keys...).Result(); err != nil {
			return err
		}
		// Loop over entries.
		for _, iface := range res {
			if strVeh, ok = iface.(string); !ok {
				continue
			}
			if err = veh.Unmarshal(strVeh); err != nil {
				return err
			}
			// @TODO: We unmarshal the vehicle before checking if
			// it satisfies the query, which is probably
			// a bit expensive. Alternatives?
			if !pq.validates(veh) {
				continue
			}
			for i, prop := range veh.Slice() {
				props[i] = prop
			}
			fmt.Fprintf(w, "%q,%q,%q,%q,%q,%q,%q,%q,%q,%q\n", props...)
			progress++
		}
		if cur == 0 || (q.Limit > 0 && progress >= q.Limit) {
			break
		}
	}
	return nil
}
