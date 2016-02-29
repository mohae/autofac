package autofac

import (
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// Client is anything that talks to the server.
type Client struct {
	ID         uint32   `json:"id"`
	Datacenter string   `json:"datacenter"`
	Groups     []string `json:"groups"`
	Roles      []string `json:"roles"`
	ServerURL  url.URL
	WS         *websocket.Conn `json:"-"`
	// channel for outbound messages
	Send       chan Message  `json:"-"`
	PingPeriod time.Duration `json:"-"`
	PongWait   time.Duration `json:"-"`
	WriteWait  time.Duration `json:"-"`
}

func NewClient(id uint32) *Client {
	return &Client{
		ID:         id,
		PingPeriod: PingPeriod,
		PongWait:   PongWait,
		WriteWait:  WriteWait,
		Send:       make(chan Message, 10),
	}
}

func (c *Client) DialServer() error {
	var err error
	c.WS, _, err = websocket.DefaultDialer.Dial(c.ServerURL.String(), nil)
	return err
}

func (c *Client) Close() error {
	return c.WS.Close()
}
