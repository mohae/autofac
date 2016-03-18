//go:generate stringer -type=Kind
package message

// Kind is the kind of message.Message.  This is used for routing.
type Kind int16

const (
	Unknown Kind = iota
	EOT          // end of transmission (for sequences that involve multiple messages, e.g. handshake)
	Generic
	Command
	SysInf
	ClientCfg
	CPUData
	MemData
)

// Int16 is a convenience method that returns the Kind as an int16 value.
// This could also be accomplished by doing the conversion directly.
func (k Kind) Int16() int16 {
	return int16(k)
}
