// automatically generated, do not modify

package sysinfo

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type MemData struct {
	_tab flatbuffers.Table
}

func GetRootAsMemData(buf []byte, offset flatbuffers.UOffsetT) *MemData {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &MemData{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *MemData) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *MemData) Timestamp() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *MemData) RAMTotal() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) RAMUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) RAMFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) RAMShared() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) RAMBuffers() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) RAMCached() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(16))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) CacheUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(18))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) CacheFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(20))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapTotal() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(22))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(24))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(26))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func MemDataStart(builder *flatbuffers.Builder) { builder.StartObject(12) }
func MemDataAddTimestamp(builder *flatbuffers.Builder, Timestamp flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(Timestamp), 0) }
func MemDataAddRAMTotal(builder *flatbuffers.Builder, RAMTotal int64) { builder.PrependInt64Slot(1, RAMTotal, 0) }
func MemDataAddRAMUsed(builder *flatbuffers.Builder, RAMUsed int64) { builder.PrependInt64Slot(2, RAMUsed, 0) }
func MemDataAddRAMFree(builder *flatbuffers.Builder, RAMFree int64) { builder.PrependInt64Slot(3, RAMFree, 0) }
func MemDataAddRAMShared(builder *flatbuffers.Builder, RAMShared int64) { builder.PrependInt64Slot(4, RAMShared, 0) }
func MemDataAddRAMBuffers(builder *flatbuffers.Builder, RAMBuffers int64) { builder.PrependInt64Slot(5, RAMBuffers, 0) }
func MemDataAddRAMCached(builder *flatbuffers.Builder, RAMCached int64) { builder.PrependInt64Slot(6, RAMCached, 0) }
func MemDataAddCacheUsed(builder *flatbuffers.Builder, CacheUsed int64) { builder.PrependInt64Slot(7, CacheUsed, 0) }
func MemDataAddCacheFree(builder *flatbuffers.Builder, CacheFree int64) { builder.PrependInt64Slot(8, CacheFree, 0) }
func MemDataAddSwapTotal(builder *flatbuffers.Builder, SwapTotal int64) { builder.PrependInt64Slot(9, SwapTotal, 0) }
func MemDataAddSwapUsed(builder *flatbuffers.Builder, SwapUsed int64) { builder.PrependInt64Slot(10, SwapUsed, 0) }
func MemDataAddSwapFree(builder *flatbuffers.Builder, SwapFree int64) { builder.PrependInt64Slot(11, SwapFree, 0) }
func MemDataEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
