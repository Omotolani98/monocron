package config

import (
		"github.com/robfig/cron/v3"
)

var Cron *cron.Cron
const SocketFile = "/var/run/monocron.sock"
const TempFile = "/tmp/monocron.sock"

