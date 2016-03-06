// automatically generated, do not modify

package client

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type ClientConf struct {
	_tab flatbuffers.Table
}

func GetRootAsClientConf(buf []byte, offset flatbuffers.UOffsetT) *ClientConf {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ClientConf{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *ClientConf) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ClientConf) HealthbeatInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ClientConf) HealthbeatPushPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ClientConf) PingPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ClientConf) PongWait() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ClientConf) SaveInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func ClientConfStart(builder *flatbuffers.Builder) { builder.StartObject(5) }
func ClientConfAddHealthbeatInterval(builder *flatbuffers.Builder, HealthbeatInterval int64) { builder.PrependInt64Slot(0, HealthbeatInterval, 0) }
func ClientConfAddHealthbeatPushPeriod(builder *flatbuffers.Builder, HealthbeatPushPeriod int64) { builder.PrependInt64Slot(1, HealthbeatPushPeriod, 0) }
func ClientConfAddPingPeriod(builder *flatbuffers.Builder, PingPeriod int64) { builder.PrependInt64Slot(2, PingPeriod, 0) }
func ClientConfAddPongWait(builder *flatbuffers.Builder, PongWait int64) { builder.PrependInt64Slot(3, PongWait, 0) }
func ClientConfAddSaveInterval(builder *flatbuffers.Builder, SaveInterval int64) { builder.PrependInt64Slot(4, SaveInterval, 0) }
func ClientConfEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
