package main

import (

	//"time"

	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofac"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func connHandler(c *autofac.Client, doneCh chan struct{}) {
	defer c.WS.Close()
	// Send the client's ID; if it's empty or can't be found, the server will
	// respond with one
	var err error
	if c.ID > 0 {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, c.ID)
		err = c.WS.WriteMessage(websocket.BinaryMessage, b)
	} else {
		var b []byte
		err = c.WS.WriteMessage(websocket.BinaryMessage, b)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while sending ID: %s\n", err)
		close(doneCh)
		return
	}
	typ, p, err := c.WS.ReadMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while Reading ID response: %s\n", err)
		close(doneCh)
		return
	}
	fmt.Printf("hello response: %d: %v\n", typ, p)
	switch typ {
	case websocket.BinaryMessage:
		// a binary response is a clientID
		fmt.Printf("ID len: %d\t%X\n", len(p), p[:4])
		c.ID = binary.LittleEndian.Uint32(p[:4])
		fmt.Printf("new ID: %d\n", c.ID)
	case websocket.TextMessage:
		fmt.Printf("%s\n", string(p))
	default:
		fmt.Printf("unexpected welcome response type %d: %v", typ, p)
	}

	go messageReader(c, doneCh)

	go messageWriter(c, doneCh)

	<-doneCh
}

func messageReader(c *autofac.Client, doneCh chan struct{}) {
	defer close(doneCh)
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s", err)
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
		case websocket.BinaryMessage:
			fmt.Printf("Binarymessage: %x\n", p)
		case websocket.PingMessage:
			fmt.Printf("pingmessage: %x\n", p)
		case websocket.PongMessage:
			fmt.Printf("pongmessage: %x\n", p)
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			return
		}
	}
}

func messageWriter(c *autofac.Client, doneCh chan struct{}) {
	defer close(doneCh)
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.WS.WriteMessage(msg.Type, msg.Data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing message: %s", err)
				return
			}
		case <-time.After(c.PingPeriod):
			fmt.Println("send ping message")
			err := c.WS.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "ping error: %s", err)
				return
			}
		}
	}
}
