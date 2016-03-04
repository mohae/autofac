package main

import (
	"time"

	"github.com/mohae/autofact"
	"github.com/mohae/autofact/db"
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
	DB db.Bolt
}

func newServer(id uint32) server {
	return server{
		ID:           id,
		PingInterval: autofact.PingPeriod,
		PongWait:     autofact.PongWait,
		Inventory:    newInventory(),
	}
}

// LoadInventory populates the inventory from the database.  This is a cached
// list of clients we are aware of.
func (s *server) LoadInventory() (int, error) {
	var n int
	ids, err := s.DB.ClientIDs()
	if err != nil {
		return n, err
	}
	for i, id := range ids {
		c := autofact.NewClient(id)
		s.Inventory.AddClient(id, c)
		n = i
	}
	return n, nil
}

// Client checks the inventory to see if the client exists
func (s *server) Client(id uint32) (*autofact.Client, bool) {
	return s.Inventory.Client(id)
}

// NewClient creates a new client.
func (s *server) NewClient() (*autofact.Client, error) {
	// get a new client
	cl := s.Inventory.NewClient()
	// save the client info to the db
	err := s.DB.SaveClient(cl.Cfg.ID)
	return cl, err
}
