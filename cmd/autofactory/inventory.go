package main

import (
	"sync"

	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/util"
)

// inventory holds a list of all the clients that the server knows of.
type inventory struct {
	nodes map[uint32]*cfg.SysInf
	mu    sync.Mutex
}

func newInventory() inventory {
	return inventory{
		nodes: map[uint32]*cfg.SysInf{},
	}
}

// AddNodeInf adds the received cfg.SysInf to the inventory.
func (i *inventory) AddNodeInf(id uint32, c *cfg.SysInf) {
	// should collision detection be done/force update cfg.SysInf
	// if it exists?
	i.mu.Lock()
	i.nodes[id] = c
	i.mu.Unlock()
}

// SaveNodeInf updates the inventory with the cfg.SysInf and saves it
// to the database.
func (i *inventory) SaveNodeInf(c *cfg.SysInf, p []byte) error {
	i.mu.Lock()
	i.nodes[c.ID()] = c
	i.mu.Unlock()
	return srvr.DB.SaveSysInf(c)
}

// NodeExists returns whether or not the node is currently in the
// inventory.  This function handles the locking and calls the unexported
// nodeExists for the actual lookup.
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

// SysInf returns the cfg.SysInf for the requested ID and wither or not
// the ID was found in the inventory.
func (i *inventory) SysInf(id uint32) (*cfg.SysInf, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	c, ok := i.nodes[id]
	return c, ok
}

// NewNode returns a new Node.  The Node will have its ID set to a
// unique value.
func (i *inventory) NewNode() *Node {
	i.mu.Lock()
	defer i.mu.Unlock()
	for {
		id := util.RandUint32()
		if !i.nodeExists(id) {
			n := newNode(id)
			i.nodes[id] = n.SysInf
			return n
		}
	}
}
