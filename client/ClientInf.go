// automatically generated, do not modify

package client

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type ClientInf struct {
	_tab flatbuffers.Table
}

func GetRootAsClientInf(buf []byte, offset flatbuffers.UOffsetT) *ClientInf {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ClientInf{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *ClientInf) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ClientInf) ID() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ClientInf) Hostname() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func ClientInfStart(builder *flatbuffers.Builder) { builder.StartObject(2) }
func ClientInfAddID(builder *flatbuffers.Builder, ID uint32) { builder.PrependUint32Slot(0, ID, 0) }
func ClientInfAddHostname(builder *flatbuffers.Builder, Hostname flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(Hostname), 0) }
func ClientInfEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
