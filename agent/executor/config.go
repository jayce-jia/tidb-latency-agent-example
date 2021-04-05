package executor

import "time"

type ConfigHolder struct {
	Latency     time.Duration
	ApplyPeriod time.Duration
}
