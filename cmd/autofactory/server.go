package main

import (
    "time"

    "github.com/mohae/autofac"
)

type Server struct {
        // ID of the server
        ID uint32
        // Interval between pings
        PingInterval time.Duration
        // How long to wait for a pong response before timing out
        PongWait time.Duration
        // A map of clients, by ID
        Clients map[uint32]*autofac.Client
}

func NewServer(id uint32) Server {
    return Server{
        ID: id,
        PingInterval: defaultPingInterval,
        PongWait: defaultPongWait,
        Clients: map[uint32]*autofac.Client{},
    }
}
