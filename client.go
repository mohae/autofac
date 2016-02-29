package autofac

import (
	"fmt"
	"net/url"
	"os"
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

func (c *Client) Listen(doneCh chan struct{}) {
	// loop until there's a done signal
	defer close(doneCh)
	for {
		fmt.Println("read message")
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
			err := c.WS.WriteMessage(typ, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing text message: %s\n", err)
				return
			}
		case websocket.BinaryMessage:
			fmt.Printf("Binarymessage: %x\n", p)
			err := c.WS.WriteMessage(typ, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing binary message: %s\n", err)
				return
			}
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			return
		}
	}
}

func (c *Client) PingHandler(msg string) error {
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("pong"))
}

func (c *Client) PongHandler(msg string) error {
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("ping"))
}
