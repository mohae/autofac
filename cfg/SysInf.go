// automatically generated, do not modify

package cfg

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type SysInf struct {
	_tab flatbuffers.Table
}

func GetRootAsSysInf(buf []byte, offset flatbuffers.UOffsetT) *SysInf {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &SysInf{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *SysInf) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *SysInf) ID() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *SysInf) Hostname() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *SysInf) Region() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *SysInf) Zone() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *SysInf) DataCenter() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func SysInfStart(builder *flatbuffers.Builder) { builder.StartObject(5) }
func SysInfAddID(builder *flatbuffers.Builder, ID uint32) { builder.PrependUint32Slot(0, ID, 0) }
func SysInfAddHostname(builder *flatbuffers.Builder, Hostname flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(Hostname), 0) }
func SysInfAddRegion(builder *flatbuffers.Builder, Region flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(Region), 0) }
func SysInfAddZone(builder *flatbuffers.Builder, Zone flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(Zone), 0) }
func SysInfAddDataCenter(builder *flatbuffers.Builder, DataCenter flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(4, flatbuffers.UOffsetT(DataCenter), 0) }
func SysInfEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
