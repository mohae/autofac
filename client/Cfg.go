// automatically generated, do not modify

package client

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type Cfg struct {
	_tab flatbuffers.Table
}

func GetRootAsCfg(buf []byte, offset flatbuffers.UOffsetT) *Cfg {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Cfg{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *Cfg) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Cfg) HealthbeatInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Cfg) HealthbeatPushPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Cfg) PingPeriod() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Cfg) PongWait() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Cfg) SaveInterval() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func CfgStart(builder *flatbuffers.Builder) { builder.StartObject(5) }
func CfgAddHealthbeatInterval(builder *flatbuffers.Builder, HealthbeatInterval int64) { builder.PrependInt64Slot(0, HealthbeatInterval, 0) }
func CfgAddHealthbeatPushPeriod(builder *flatbuffers.Builder, HealthbeatPushPeriod int64) { builder.PrependInt64Slot(1, HealthbeatPushPeriod, 0) }
func CfgAddPingPeriod(builder *flatbuffers.Builder, PingPeriod int64) { builder.PrependInt64Slot(2, PingPeriod, 0) }
func CfgAddPongWait(builder *flatbuffers.Builder, PongWait int64) { builder.PrependInt64Slot(3, PongWait, 0) }
func CfgAddSaveInterval(builder *flatbuffers.Builder, SaveInterval int64) { builder.PrependInt64Slot(4, SaveInterval, 0) }
func CfgEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
