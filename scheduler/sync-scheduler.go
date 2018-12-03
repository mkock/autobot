package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mkock/autobot/dataprovider"
	"github.com/mkock/autobot/dmr"

	"github.com/gorhill/cronexpr"
	"github.com/mkock/autobot/config"
	"github.com/mkock/autobot/vehicle"
)

// SyncScheduler represents a new scheduler.
type SyncScheduler struct {
	cnf       config.Config
	store     *vehicle.Store
	schedExpr *cronexpr.Expression
}

// New returns a new scheduler that schedules and runs data synchronisation with the vehicle store
// on fixed intervals defined by the provided configuration. For scheduling, the common cron time expression syntax
// is used, but in the five-field variant where the fields have the following interpretation:
// "minute hours day-of-month month day-of-week"
func New(cnf config.Config, store *vehicle.Store) *SyncScheduler {
	return &SyncScheduler{cnf, store, nil}
}

// parseTimeExpr parses the schedule given in the Config and assigns a parsed (cron-style) time expression
// to the scheduler.
func (sched *SyncScheduler) parseTimeExpr() error {
	if strings.Count(sched.cnf.WebService.Schedule, " ") != 4 {
		return fmt.Errorf("invalid expression: %s, must be a five-field cron-styled time expression", sched.cnf.WebService.Schedule)
	}
	sched.schedExpr = cronexpr.MustParse(sched.cnf.WebService.Schedule)
	return nil
}

// Start starts the scheduler. It will run forever until interrupted.
// It returns a channel that you can send a bool on in order to interrupt the scheduler and shut it down gracefully.
func (sched *SyncScheduler) Start() (chan<- bool, error) {
	if err := sched.parseTimeExpr(); err != nil {
		return nil, err
	}
	stop := make(chan bool)
	go sched.repeatSync(stop)
	return stop, nil
}

// repeatSync listens on channel "stop", and until it receives a stop signal (a boolean value) on this channel,
// it will keep calculating the next "tick", sleep until that time, run the synchronisation job and repeat.
func (sched *SyncScheduler) repeatSync(stop <-chan bool) {
	var (
		now, next time.Time
		dur       time.Duration
	)
	for {
		now = time.Now()
		next = sched.schedExpr.Next(now)
		dur = next.Sub(now)
		if int64(dur) < 0 {
			fmt.Println("Invalid duration") // This should never happen, but let's check.
			return
		}
		select {
		case <-stop:
			return
		case _ = <-time.After(dur):
			// The call to doSync is synchronous so we don't risk starting several sync jobs on top of each other.
			if err := sched.doSync(); err != nil {
				log.Printf("Sync error: %s, will retry later", err)
			}
		}
	}
}

// doSync starts the actual data synchronisation.
// Note: it currently only supports synchronisation with DMR.
func (sched *SyncScheduler) doSync() error {
	var (
		fname, latest string
		err           error
	)
	fname, _ = sched.store.GetLastSynced()
	prov := dataprovider.NewProvider(dataprovider.FtpProv, sched.cnf.Providers["DMR"])
	if err = prov.Open(); err != nil {
		return err
	}
	latest, err = prov.CheckForLatest(fname)
	if err != nil {
		return err
	}
	if latest == "" || latest == fname {
		log.Print("Sync: No new stat file detected")
		return nil
	}
	log.Printf("Sync: synchronising %s from DMR...\n", latest)
	src, err := prov.Provide(latest)
	if err != nil {
		return err
	}
	if src == nil {
		log.Println("Sync: no stat file detected. Aborting")
		return nil
	}
	id := sched.store.NewSyncOp(dataprovider.ProvTypeString(dataprovider.FtpProv))

	dmrService := dmr.NewService()
	vehicles, done := dmrService.LoadNew(src)
	if err = sched.store.Sync(id, vehicles, done); err != nil {
		return err
	}
	fmt.Println(sched.store.Status(id))

	if err = sched.store.SetLastSynced(latest); err != nil {
		return err
	}
	return nil
}
