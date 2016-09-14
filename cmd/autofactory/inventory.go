package main

import (
	"sync"

	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/util"
)

// inventory holds a list of all the Nodes that the server knows of.
type inventory struct {
	nodes map[uint32]*cfg.Node
	mu    sync.Mutex
}

func newInventory() inventory {
	return inventory{
		nodes: map[uint32]*cfg.Node{},
	}
}

// AddNode adds the received *cfg.Node to the inventory.
func (i *inventory) AddNode(id uint32, c *cfg.Node) {
	// TODO: should collision detection be done/force update cfg.Node
	// if it exists? or generate an error?
	i.mu.Lock()
	i.nodes[id] = c
	i.mu.Unlock()
}

// SaveNode updates the inventory with the cfg.Node and saves it to the
// database.
func (i *inventory) SaveNode(c *cfg.Node, p []byte) error {
	i.mu.Lock()
	i.nodes[c.ID()] = c
	i.mu.Unlock()
	return srvr.DB.SaveNode(c)
}

// NodeExists returns whether or not the node is currently in the inventory.
// This function handles the locking and calls the unexported nodeExists for
// the actual lookup.
func (i *inventory) NodeExists(id uint32) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.nodeExists(id)
}

// nodeExists returns whether or not the requrested ID is in the inventory.
// This does not do any locking because there are methods that already have
// a lock on the inventory that need to check for existence.  External callers
// should use NodeExists().
func (i *inventory) nodeExists(id uint32) bool {
	_, ok := i.nodes[id]
	return ok
}

// Node returns the *cfg.Node for the requested ID and wither or not the ID
// was found in the inventory.
func (i *inventory) Node(id uint32) (*cfg.Node, bool) {
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
			i.nodes[id] = n.Node
			return n
		}
	}
}
