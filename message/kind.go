//go:generate stringer -type=Kind
package message

type Kind int

const (
	Unknown Kind = iota
	Generic
	Command
	CPUStat
)
