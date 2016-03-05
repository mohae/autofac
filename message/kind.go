//go:generate stringer -type=Kind
package message

type Kind int16

const (
	Unknown Kind = iota
	Generic
	Command
	CPUData
	MemData
)

func (k Kind) Int16() int16 {
	return int16(k)
}
