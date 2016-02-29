package main

import (
	"time"

	"github.com/mohae/autofac"
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
}

func newServer(id uint32) server {
	return server{
		ID:           id,
		PingInterval: autofac.PingPeriod,
		PongWait:     autofac.PongWait,
		Inventory:    newInventory(),
	}
}
