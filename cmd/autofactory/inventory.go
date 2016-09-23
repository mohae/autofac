package main

import (
	"sync"

	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/util"
)

// inventory holds information about all of the nodes the system knows about.
type inventory struct {
	nodes map[uint32]*cfg.NodeInf
	mu    sync.Mutex
}

func newInventory() inventory {
	return inventory{
		nodes: map[uint32]*cfg.NodeInf{},
	}
}

// AddNode adds a node's information to the inventory.
func (i *inventory) AddNode(id uint32, c *cfg.NodeInf) {
	// TODO: should collision detection be done/force update cfg.Node
	// if it exists? or generate an error?
	i.mu.Lock()
	i.nodes[id] = c
	i.mu.Unlock()
}

// SaveNode updates the inventory with a node's information and saves it to
// the database.
func (i *inventory) SaveNode(c *cfg.NodeInf, p []byte) error {
	i.mu.Lock()
	i.nodes[c.ID()] = c
	i.mu.Unlock()
	return srvr.DB.SaveNode(c)
}

// NodeExists returns whether or not a specific node is currently in the
// inventory.
func (i *inventory) NodeExists(id uint32) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.nodeExists(id)
}

// nodeExists returns whether or not the requrested ID is in the inventory.
// This does not do any locking; it is assumed that the color is properly
// managing the lock's state properly.
func (i *inventory) nodeExists(id uint32) bool {
	_, ok := i.nodes[id]
	return ok
}

// Node returns true and the information for the requested ID, if it exists,
// otherwise false is returned.
func (i *inventory) Node(id uint32) (*cfg.NodeInf, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	c, ok := i.nodes[id]
	return c, ok
}

// NewNode returns a new Node.  The Node will have its ID set to a unique
// value.
func (i *inventory) NewNode() *Client {
	i.mu.Lock()
	defer i.mu.Unlock()
	for {
		id := util.RandUint32()
		if !i.nodeExists(id) {
			n := newClient(id)
			i.nodes[id] = n.NodeInf
			return n
		}
	}
}
