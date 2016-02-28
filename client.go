package autofac

import (
	"github.com/gorilla/websocket"
)

// Client is anything that talks to the server.
type Client struct {
	ID   uint32	`json:"id"`
	Datacenter string `json:"datacenter"`
	Groups []string `json:"groups"`
	Roles []string `json:"roles"`
	ws   *websocket.Conn
	send chan Message
}

// Send sends the message to the server
func (c *Client) Write(msg Message) error {
	return c.ws.WriteMessage(msg.Type, msg.Data)
}

func (c *Client) Close() error {
	close (c.send)
	return c.ws.Close()
}

func (c *Client) Conn() *websocket.Conn {
	return c.ws
}

func NewClient(id uint32) *Client {
	return &Client{
		ID: id,
		send: make(chan Message),
	}
}
