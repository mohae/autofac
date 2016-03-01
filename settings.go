package autofact

import "time"

var (
	WriteBufferSize = 1024
	ReadBufferSize  = 1024

	WriteWait  = 5 * time.Second
	PongWait   = 30 * time.Second
	PingPeriod = 25 * time.Second
)
