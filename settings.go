package autofact

import "time"

// PathVarName is the environment variable name for the autofact path.
const PathVarName = "AUTOFACTPATH"

// Autofact defaults
var (
	DefaultPingPeriod = time.Duration(25) * time.Second
	DefaultPongWait   = time.Duration(30) * time.Second
)

var (
	// WriteBufferSize is the The default size for a websocket write buffer.
	WriteBufferSize = 1024
	// ReadBufferSize is the The default size for a websocket read buffer.
	ReadBufferSize = 1024

	// WriteWait is the default time to wait for a write to succeed.
	WriteWait = 5 * time.Second
)

// AckMsg is the default value for ack'ing received messages.
var AckMsg = []byte("ok")
