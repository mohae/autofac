// automatically generated, do not modify

package cfg

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type Client struct {
	_tab flatbuffers.Table
}

func GetRootAsClient(buf []byte, offset flatbuffers.UOffsetT) *Client {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Client{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *Client) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Client) HealthbeatInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Client) HealthbeatPushPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Client) PingPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Client) PongWait() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Client) SaveInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func ClientStart(builder *flatbuffers.Builder) { builder.StartObject(5) }
func ClientAddHealthbeatInterval(builder *flatbuffers.Builder, HealthbeatInterval int64) { builder.PrependInt64Slot(0, HealthbeatInterval, 0) }
func ClientAddHealthbeatPushPeriod(builder *flatbuffers.Builder, HealthbeatPushPeriod int64) { builder.PrependInt64Slot(1, HealthbeatPushPeriod, 0) }
func ClientAddPingPeriod(builder *flatbuffers.Builder, PingPeriod int64) { builder.PrependInt64Slot(2, PingPeriod, 0) }
func ClientAddPongWait(builder *flatbuffers.Builder, PongWait int64) { builder.PrependInt64Slot(3, PongWait, 0) }
func ClientAddSaveInterval(builder *flatbuffers.Builder, SaveInterval int64) { builder.PrependInt64Slot(4, SaveInterval, 0) }
func ClientEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
