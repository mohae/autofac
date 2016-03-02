// automatically generated, do not modify

package sysinfo

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type CPUStatFB struct {
	_tab flatbuffers.Table
}

func GetRootAsCPUStatFB(buf []byte, offset flatbuffers.UOffsetT) *CPUStatFB {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &CPUStatFB{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *CPUStatFB) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *CPUStatFB) Timestamp() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CPUStatFB) CPUID() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CPUStatFB) Usr() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Nice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Sys() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) IOWait() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) IRQ() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(16))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Soft() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(18))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Steal() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(20))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Guest() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(22))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) GNice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(24))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStatFB) Idle() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(26))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func CPUStatFBStart(builder *flatbuffers.Builder) { builder.StartObject(12) }
func CPUStatFBAddTimestamp(builder *flatbuffers.Builder, Timestamp flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(Timestamp), 0) }
func CPUStatFBAddCPUID(builder *flatbuffers.Builder, CPUID flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(CPUID), 0) }
func CPUStatFBAddUsr(builder *flatbuffers.Builder, Usr int16) { builder.PrependInt16Slot(2, Usr, 0) }
func CPUStatFBAddNice(builder *flatbuffers.Builder, Nice int16) { builder.PrependInt16Slot(3, Nice, 0) }
func CPUStatFBAddSys(builder *flatbuffers.Builder, Sys int16) { builder.PrependInt16Slot(4, Sys, 0) }
func CPUStatFBAddIOWait(builder *flatbuffers.Builder, IOWait int16) { builder.PrependInt16Slot(5, IOWait, 0) }
func CPUStatFBAddIRQ(builder *flatbuffers.Builder, IRQ int16) { builder.PrependInt16Slot(6, IRQ, 0) }
func CPUStatFBAddSoft(builder *flatbuffers.Builder, Soft int16) { builder.PrependInt16Slot(7, Soft, 0) }
func CPUStatFBAddSteal(builder *flatbuffers.Builder, Steal int16) { builder.PrependInt16Slot(8, Steal, 0) }
func CPUStatFBAddGuest(builder *flatbuffers.Builder, Guest int16) { builder.PrependInt16Slot(9, Guest, 0) }
func CPUStatFBAddGNice(builder *flatbuffers.Builder, GNice int16) { builder.PrependInt16Slot(10, GNice, 0) }
func CPUStatFBAddIdle(builder *flatbuffers.Builder, Idle int16) { builder.PrependInt16Slot(11, Idle, 0) }
func CPUStatFBEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
