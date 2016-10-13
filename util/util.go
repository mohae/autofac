package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	pcg "github.com/dgryski/go-pcgr"
)

// max value for an int64
const (
	maxInt64 = 1<<63 - 1
	alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	IDLen    = 8
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
func Uint8ToBool(v byte) bool {
	if v == 0x00 {
		return false
	}
	return true
}

// BoolToByte returns the byte equivalent of a bool.  Flatbuffers aliases bool
// to byte.
func BoolToUint8(b bool) byte {
	if b {
		return 0x01
	}
	return 0x00
}
