package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	pcg "github.com/dgryski/go-pcgr"
)

// max value for an int64
const (
	maxInt64 = 1<<63 - 1
	alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	IDLen    = 8
	Epoch    = "epoch" // constant for using the raw timestamp instead of a time.Format
)

// util has its own prng
var prng pcg.Rand

func init() {
	prng.Seed(seed())
}

// seed gets a random int64 using a CSPRNG.
func seed() int64 {
	bi := big.NewInt(maxInt64)
	r, err := rand.Int(rand.Reader, bi)
	if err != nil {
		panic(fmt.Sprintf("entropy read error: %s", err))
	}
	return r.Int64()
}

// ReSeed reseeds the PRNG using a value from crypto.Rand.
func ReSeedPRNG() {
	prng.Seed(seed())
}

// NewStringID returns a randomly generated string of length n.  This string is not
// guaranteed to be uniqe.  The caller should do a collision check.
func NewStringID(n int) string {
	id := make([]byte, n)
	for i := 0; i < n; i++ {
		id[i] = alphanum[prng.Bound(uint32(len(alphanum)))]
	}
	return string(id)
}

// RandUint32 returns a uint32 obtained from prng
func RandUint32() uint32 {
	return prng.Next()
}

// Int64ToBytes takes an int64 and returns it as an 8 byte array.
func Int64ToBytes(x int64) [8]byte {
	var b [8]byte
	b[0] = byte(x >> 56)
	b[1] = byte(x >> 48)
	b[2] = byte(x >> 40)
	b[3] = byte(x >> 32)
	b[4] = byte(x >> 24)
	b[5] = byte(x >> 16)
	b[6] = byte(x >> 8)
	b[7] = byte(x)
	return b
}

// Int64ToByteSlice takes an int64 and returns it as a slice of bytes.
func Int64ToByteSlice(x int64) []byte {
	b := make([]byte, 8)
	b[0] = byte(x >> 56)
	b[1] = byte(x >> 48)
	b[2] = byte(x >> 40)
	b[3] = byte(x >> 32)
	b[4] = byte(x >> 24)
	b[5] = byte(x >> 16)
	b[6] = byte(x >> 8)
	b[7] = byte(x)
	return b[:]
}

// Duration is an alias for time.Duration
type Duration struct {
	time.Duration
}

// MarshalJSON marshal's the duration as a string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// UnmarshalJSON unmarshals a string as a Duration.  If the value doesn't start
// with a quote, ", an error will occur.  See time.ParseDuration for valid
// values.
func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	// If this is a string; parse it as such.
	if b[0] != '"' {
		return fmt.Errorf("duration: unable to UnmarshalJSON not a string: %v", b)
	}
	d.Duration, err = time.ParseDuration(string(b[1 : len(b)-1]))
	return err
}

// Int64 is a helper func that returns the duration as an int64.
func (d *Duration) Int64() int64 {
	return int64(d.Duration)
}

// Set is a helper function that sets the duration with the value.
func (d *Duration) Set(v int64) {
	d.Duration = time.Duration(v)
}

// WSString returns a string for a given websocket message type
func WSString(t int) string {
	switch t {
	case 1:
		return "TextMessage"
	case 2:
		return "BinaryMessage"
	case 8:
		return "CloseMessage"
	case 9:
		return "PingMessage"
	case 10:
		return "PongMessage"
	default:
		return "UnknownMessage"
	}
}

// ByteToBool returns the bool equivalent of a byte.  Flatbuffers aliases bool
// to byte.
func ByteToBool(v byte) bool {
	if v == 0x00 {
		return false
	}
	return true
}

// BoolToByte returns the byte equivalent of a bool.  Flatbuffers aliases bool
// to byte.
func BoolToByte(b bool) byte {
	if b {
		return 0x01
	}
	return 0x00
}

// TimeLayout set's the data output's time layout.  This handles any layout
// that is a constant as defined by time.Constants along with ts or timestamp.
// The default is 'epoch', if the layout string is either empty, or is 'epoch'
// the time will be written out as time since unix epoch.
//
// If the specified time format is not a string that matches a time.Constants
// layout, it will be asumed that the specified format is valid; no validation
// will be done on the specified format.
//
// All input is upper cased prior to evaluation.
func TimeLayout(l string) string {
	if len(l) == 0 {
		return Epoch
	}
	// uppercase the layout for consistency
	l = strings.ToUpper(l)
	switch l {
	case "EPOCH":
		return Epoch // this is the only value that doesn't get formatted
	case "ANSIC":
		return time.ANSIC
	case "UNIXDATE":
		return time.UnixDate
	case "RUBYDATE":
		return time.RubyDate
	case "RFC822":
		return time.RFC822
	case "RFC822Z":
		return time.RFC822Z
	case "RFC850":
		return time.RFC850
	case "RFC1123":
		return time.RFC1123
	case "RFC1123Z":
		return time.RFC1123Z
	case "RFC3339":
		return time.RFC3339
	case "RFC3339Nano":
		return time.RFC3339Nano
	case "KITCHEN":
		return time.Kitchen
	case "STAMP":
		return time.Stamp
	case "STAMPMILLI":
		return time.StampMilli
	case "STAMPMICRO":
		return time.StampMicro
	case "STAMPNANO":
		return time.StampNano
	default:
		return l
	}
}
