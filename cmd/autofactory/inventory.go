package main

import (
	"sync"

	"github.com/mohae/autofact/conf"
)

// inventory holds information about all of the nodes the system knows about.
type inventory struct {
	clients map[string]*conf.Client
	mu      sync.Mutex
}

func newInventory() inventory {
	return inventory{
		clients: make(map[string]*conf.Client),
	}
}

// AddNode adds a client's information to the inventory.
func (i *inventory) AddClient(id string, c *conf.Client) {
	// TODO: should collision detection be done/force update cfg.Client
	// if it exists? or generate an error?
	i.mu.Lock()
	i.clients[id] = c
	i.mu.Unlock()
}

// SaveClient updates the inventory with a client's information and saves it to
// the database.
func (i *inventory) SaveClient(c *conf.Client, p []byte) error {
	i.mu.Lock()
	i.clients[string(c.ID())] = c
	i.mu.Unlock()
	return srvr.DB.SaveClient(c)
}

// ClientExists returns whether or not a specific client is currently in the
// inventory.
func (i *inventory) ClientExists(id string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.clientExists(id)
}

// clientExists returns whether or not the requrested ID is in the inventory.
// This does not do any locking; it is assumed that the color is properly
// managing the lock's state properly.
func (i *inventory) clientExists(id string) bool {
	_, ok := i.clients[id]
	return ok
}

// Client returns true and the information for the requested ID, if it exists,
// otherwise false is returned.
func (i *inventory) Client(id string) (*conf.Client, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	c, ok := i.clients[id]
	return c, ok
}

// NewClient returns a new Client.  The Node will have its ID set to a unique
// value.
func (i *inventory) NewClient() *Client {
	i.mu.Lock()
	defer i.mu.Unlock()
	for {
		// TOD replace with a rand bytes or striing
		id := ""
		if !i.clientExists(id) {
			c := newClient(id)
			i.clients[id] = c.Conf
			return c
		}
	}
}
