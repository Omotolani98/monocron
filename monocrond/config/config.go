package config

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

const SocketFile = "/run/monocron.sock"
const TempFile = "/tmp/monocron.sock"

var (
	Cron = cron.New(
		cron.WithSeconds(),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)

	Mu   sync.RWMutex
	Jobs = map[cron.EntryID]*JobMeta{}
)

type JobMeta struct {
	Name      string
	Spec      string
	CreatedAt time.Time
	Timeout   time.Duration
}

func StartCron() { Cron.Start() }
