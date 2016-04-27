package client

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	cpuutil "github.com/mohae/joefriday/cpu/utilization/flat"
	netf "github.com/mohae/joefriday/net/usage/flat"
	loadf "github.com/mohae/joefriday/sysinfo/load/flat"
	memf "github.com/mohae/joefriday/sysinfo/mem/flat"
)

// Client is anything that talks to the server.
type Client struct {
	// The Autofact Path
	AutoPath string
	// SysInf holds the basic client information.
	*cfg.SysInf
	// This is SysInf as bytes: this is hopefully unnecessary, but I don't know at this point.
	SysInfBytes []byte
	// Conn holds the configuration for connecting to the server.
	cfg.Conn
	// Cfg holds the client configuration (how the client behaves).
	Cfg *cfg.Client
	// DB conn for clients
	DB db.Bolt

	// queue of healthbeat messages to be sent.
	healthbeatQ message.Queue

	// this lock is for everything except messages or other things that are
	// already threadsafe.
	mu sync.Mutex
	// The websocket connection that this client uses.
	WS *websocket.Conn
	// Channel for outbound binary messages.  The message is assumed to be a
	// websocket.Binary type
	SendB       chan []byte
	SendStr     chan string
	isConnected bool
	ServerURL   url.URL
}

func New(inf *cfg.SysInf) *Client {
	return &Client{
		SysInf:      inf,
		SysInfBytes: inf.Serialize(),
		healthbeatQ: message.NewQueue(32), // this is just an arbitrary number. TODO revisit.
		// A really small buffer:
		// TODO: rethink this vis-a-vis what happens when recipient isn't there
		// or if it goes away during sending and possibly caching items to be sent.
		SendB:   make(chan []byte, 10),
		SendStr: make(chan string, 10),
	}
}

// Connect handles connecting to the server and returns the connection status.
// The client will attempt to connect until it has either succeeded or the
// connection retry period has been exceeded.  A retry is done every 5 seconds.
//
// If the client is already connected, nothing will be done.
func (c *Client) Connect() bool {
	// If already connected, return that fact.
	if c.IsConnected() {
		return true
	}
	start := time.Now()
	retryEnd := start.Add(c.ConnectPeriod)
	// connect to server; retry until the retry period has expired
	for {
		if time.Now().After(retryEnd) {
			fmt.Fprintf(os.Stderr, "timed out while trying to connect to the server: %s\n", c.ServerURL.String())
			return false
		}
		err := c.DialServer()
		if err == nil {
			break
		}
		time.Sleep(c.ConnectInterval)
		fmt.Printf("unable to connect to the server %s: retrying...\n", c.ServerURL.String())
	}
	// Send the ClientInf.  If the ID == 0 or it can't be found, the server will
	// respond with one.  Retry until the server responds, or until the
	// reconnectPeriod has expired.
	err := c.WS.WriteMessage(websocket.BinaryMessage, c.SysInfBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while sending ID: %s\n", err)
		c.WS.Close()
		return false
	}

	// read messages until we get an EOT
handshake:
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while Reading ID response: %s\n", err)
			c.WS.Close()
			return false
		}

		switch typ {
		case websocket.BinaryMessage:
			// process according to message kind
			msg := message.GetRootAsMessage(p, 0)
			switch message.Kind(msg.Kind()) {
			case message.SysInf:
				c.SysInf = cfg.GetRootAsSysInf(msg.DataBytes(), 0)
			case message.ClientCfg:
				c.Cfg = cfg.GetRootAsClient(msg.DataBytes(), 0)
			case message.EOT:
				break handshake
			default:
				fmt.Fprint(os.Stderr, "unknown message type received during handshake; quitting\n")
				return false
			}
		case websocket.TextMessage:
			fmt.Printf("%s\n", string(p))
		default:
			fmt.Printf("unexpected welcome response type %d: %v\n", typ, p)
			c.WS.Close()
			return false
		}
	}
	fmt.Printf("%X connected\n", c.SysInf.ID())
	c.mu.Lock()
	c.isConnected = true
	c.mu.Unlock()
	return true
}

func (c *Client) DialServer() error {
	var err error
	c.WS, _, err = websocket.DefaultDialer.Dial(c.ServerURL.String(), nil)
	return err
}

func (c *Client) MessageWriter(doneCh chan struct{}) {
	pingPeriod := time.Duration(c.Cfg.PingPeriod())
	defer close(doneCh)
	for {
		select {
		case p, ok := <-c.SendB:
			// don't send if not connected
			if !c.IsConnected() {
				// TODO add write to db for persistence instead.
				continue
			}
			if !ok {
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.WS.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing message: %s\n", err)
			}
		case <-time.After(pingPeriod):
			// only ping if we are connected
			if !c.IsConnected() {
				continue
			}
			err := c.WS.WriteMessage(websocket.PingMessage, []byte("ping"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "ping error: %s\n", err)
			}
		}
	}
}

func (c *Client) Reconnect() bool {
	c.mu.Lock()
	c.isConnected = false
	c.mu.Unlock()
	for i := 0; i < 4; i++ {
		b := c.Connect()
		if b {
			fmt.Println("reconnect true")
			return b
		}
	}
	return false
}

func (c *Client) Listen(doneCh chan struct{}) {
	// loop until there's a done signal
	defer close(doneCh)
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
			if _, ok := err.(*websocket.CloseError); !ok {
				return
			}
			fmt.Println("reconnecting from read messages")
			connected := c.Reconnect()
			if connected {
				continue
			}
			fmt.Fprintln(os.Stderr, "unable to reconnect to server")
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
			if bytes.Equal(p, autofact.AckMsg) {
				// if this is an acknowledgement message, do nothing
				// TODO: should tracking of acks, per message, for certain
				// message kinds be done?
				continue
			}
			err := c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("reconnect from writing message: textmessage")
				connected := c.Reconnect()
				if connected {
					continue
				}
				fmt.Fprintln(os.Stderr, "unable to reconnect to server")
				return
			}
		case websocket.BinaryMessage:
			err = c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing binary message: %s\n", err)
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("reconnect from writing message: binarymessage")
				connected := c.Reconnect()
				if connected {
					continue
				}
				fmt.Fprintln(os.Stderr, "unable to reconnect to server")
				return
			}
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			fmt.Println("reconnect from writing message: closemessage")
			connected := c.Reconnect()
			if connected {
				continue
			}
			fmt.Fprintln(os.Stderr, "unable to reconnect to server")
			return
		}
	}
}

// IsConnected returns if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *Client) PingHandler(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("ping"))
}

func (c *Client) PongHandler(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("pong"))
}

// Healthbeat gathers basic system stats at a given interval.
// TODO: handle doneCh stuff
// TODO: HealthBeatPushPeriod vs sending message as they are
// generated.
func (c *Client) Healthbeat() {
	// An interval of 0 means no healthbeat
	if c.Cfg.HealthbeatInterval() == 0 {
		return
	}
	// error channel
	errCh := make(chan error)
	// ticker for cpu utilization data
	cpuTicker, err := cpuutil.NewTicker(time.Duration(c.Cfg.HealthbeatInterval()))
	if err != nil {
		errCh <- err
		return
	}
	cpuTickr := cpuTicker.(*cpuutil.Ticker)
	// make sure the resources get cleaned up
	defer cpuTickr.Close()
	defer cpuTickr.Stop()
	// ticker for loadavg data
	loadTicker, err := loadf.NewTicker(time.Duration(c.Cfg.HealthbeatInterval()))
	if err != nil {
		errCh <- err
		return
	}
	loadTickr := loadTicker.(*loadf.Ticker)
	// make sure the resources get cleaned up
	defer loadTickr.Close()
	defer loadTickr.Stop()
	// ticker for meminfo data
	memTicker, err := memf.NewTicker(time.Duration(c.Cfg.HealthbeatInterval()))
	if err != nil {
		errCh <- err
		return
	}
	memTickr := memTicker.(*memf.Ticker)
	// make sure the resources get cleaned up
	defer memTickr.Close()
	defer memTickr.Stop()
	// ticker for network usage data
	netTicker, err := netf.NewTicker(time.Duration(c.Cfg.HealthbeatInterval()))
	if err != nil {
		errCh <- err
		return
	}
	netTickr := netTicker.(*netf.Ticker)
	// make sure the resources get cleaned up
	defer netTickr.Close()
	defer netTickr.Stop()
	//	go mem.DataTicker(time.Duration(c.Cfg.HealthbeatInterval()), memCh, doneCh, errCh)
	//	go net.DataTicker(time.Duration(c.Cfg.HealthbeatInterval()), netdevCh, doneCh, errCh)
	t := time.NewTicker(time.Duration(c.Cfg.HealthbeatPushPeriod()))
	defer t.Stop()
	for {
		select {
		case data, ok := <-cpuTickr.Data:
			if !ok {
				fmt.Println("cpu utilization chan closed")
				goto done
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.CPUUtilization, data})
		case err := <-cpuTickr.Errs:
			fmt.Fprintln(os.Stderr, err)
		case data, ok := <-loadTickr.Data:
			if !ok {
				fmt.Println("load avg chan closed")
				goto done
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.SysLoadAvg, data})
		case err := <-loadTickr.Errs:
			fmt.Fprintln(os.Stderr, err)
		case data, ok := <-memTickr.Data:
			if !ok {
				fmt.Println("mem info chan closed")
				goto done
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.SysMemInfo, data})
		case err := <-memTickr.Errs:
			fmt.Fprintln(os.Stderr, err)
		case data, ok := <-netTickr.Data:
			if !ok {
				fmt.Println("network usage stats chan closed")
				goto done
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.NetUsage, data})
		case err := <-netTickr.Errs:
			fmt.Fprintln(os.Stderr, err)
		case <-t.C:
			c.SendHealthbeatMessages()
		}
	}
done:
	// Flush the queue.
	c.SendHealthbeatMessages()
}

// SendHealthbeatMessages sends everything in the healthbeat queue.
// TODO:  should this be more resilient?
func (c *Client) SendHealthbeatMessages() error {
	// for each data, send a message
	for {
		m, ok := c.healthbeatQ.Dequeue()
		if !ok { // nothing left to send
			break
		}
		c.SendB <- message.Serialize(c.SysInf.ID(), m.Kind, m.Data)
	}
	return nil
}

// SendMessage sends a single serialized message of type Kind.
func (c *Client) SendMessage(kind message.Kind, p []byte) {
	c.SendB <- message.Serialize(c.SysInf.ID(), kind, p)
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.ClientCfg:
		c.Cfg.Deserialize(msg.DataBytes())
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}
