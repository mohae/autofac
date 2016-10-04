package autofact

import "time"

// PathVarName is the environment variable name for the autofact path.
const PathVarName = "AUTOFACTPATH"

var (
	// WriteBufferSize is the The default size for a websocket write buffer.
	WriteBufferSize = 1024
	// ReadBufferSize is the The default size for a websocket read buffer.
	ReadBufferSize = 1024

	// WriteWait is the default time to wait for a write to succeed.
	WriteWait = 5 * time.Second
)

// Text Message stuff.
var (
	// AckMsg is the default value for ack'ing received messages.
	AckMsg = []byte("ok")
	// LoadAvg is used for requesting a system's loadavg.
	LoadAvg = []byte("loadavg")
)
