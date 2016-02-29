package main

import (
	"sync"

	"github.com/mohae/autofac"
	"github.com/mohae/autofac/util"
)

type inventory struct {
	clients map[uint32]*autofac.Client
	mu      sync.Mutex
}

func newInventory() inventory {
	return inventory{
		clients: map[uint32]*autofac.Client{},
	}
}

func (i *inventory) AddClient(id uint32, c *autofac.Client) {
	// should collision detection be done/force update Client
	// if it exists?
	i.mu.Lock()
	i.clients[id] = c
	i.mu.Unlock()
}

func (i *inventory) ClientExists(id uint32) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.clientExists(id)
}

func (i *inventory) clientExists(id uint32) bool {
	_, ok := i.clients[id]
	return ok
}

func (i *inventory) Client(id uint32) (*autofac.Client, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	c, ok := i.clients[id]
	return c, ok
}

func (i *inventory) NewClient() *autofac.Client {
	i.mu.Lock()
	defer i.mu.Unlock()
	for {
		id := util.RandUint32()
		if !i.clientExists(id) {
			c := autofac.NewClient(id)
			i.clients[id] = c
			return c
		}
	}
}
