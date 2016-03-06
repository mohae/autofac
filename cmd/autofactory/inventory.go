package main

import (
	"sync"

	"github.com/mohae/autofact/util"
)

// inventory holds a list of all the clients that the server knows of.
type inventory struct {
	clients map[uint32]*Client
	mu      sync.Mutex
}

func newInventory() inventory {
	return inventory{
		clients: map[uint32]*Client{},
	}
}

// AddClient adds the received client to the inventory.
func (i *inventory) AddClient(id uint32, c *Client) {
	// should collision detection be done/force update Client
	// if it exists?
	i.mu.Lock()
	i.clients[id] = c
	i.mu.Unlock()
}

// ClientExists returns whether or not the client is currently in the
// inventory.  This function handles the locking and calls the unexported
// clientExists for the actual lookup.
func (i *inventory) ClientExists(id uint32) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.clientExists(id)
}

// clientExists returns whether or not the requrested ID is in the inventory.
// This does not do any locking because there are methods that already have
// a lock on the inventory that need to check for existence.  External callers
// should use ClientExists().
func (i *inventory) clientExists(id uint32) bool {
	_, ok := i.clients[id]
	return ok
}

// Client returns the Client for the requested ID and wither or not the ID
// was found in the inventory.
func (i *inventory) Client(id uint32) (*Client, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	c, ok := i.clients[id]
	return c, ok
}

// NewClient returns a new Client.  The Client will have its ID set to a
// unique value.
func (i *inventory) NewClient() *Client {
	i.mu.Lock()
	defer i.mu.Unlock()
	for {
		id := util.RandUint32()
		if !i.clientExists(id) {
			c := newClient(id)
			i.clients[id] = c
			return c
		}
	}
}
