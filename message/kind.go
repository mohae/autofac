//go:generate stringer -type=Kind
package message

type Kind int16

const (
	Unknown Kind = iota
	Generic
	Command
	CPUStat
)
