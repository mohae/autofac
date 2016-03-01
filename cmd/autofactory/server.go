package main

import (
	"time"

	"github.com/mohae/autofact"
)

type server struct {
	// ID of the server
	ID uint32
	// Interval between pings
	PingInterval time.Duration
	// How long to wait for a pong response before timing out
	PongWait time.Duration
	// A map of clients, by ID
	Inventory inventory
	// TODO: add handling to prevent the same client from connecting
	// more than once:  this requires detection of reconnect of an
	// existing client vs an existing client maintaining multiple
	// con-current connections
}

func newServer(id uint32) server {
	return server{
		ID:           id,
		PingInterval: autofact.PingPeriod,
		PongWait:     autofact.PongWait,
		Inventory:    newInventory(),
	}
}
