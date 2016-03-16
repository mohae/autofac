package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	pcg "github.com/dgryski/go-pcgr"
)

// max value for an int64
const maxInt64 = 1<<63 - 1
const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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
	b := Int64ToBytes(x)
	return b[:]
}

// Duration is an alias for time.Duration
type Duration time.Duration

// UnmarshalJSON takes a slice of bytes and converts it to a duration.  An
// error is returned if  the slice of bytes doesn't contain a value that can
// be parsed into a duration.
// returned.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*d = 0
		return nil
	}

	text := string(data)
	t, err := time.ParseDuration(text)
	if err == nil {
		*d = Duration(t)
		return nil
	}
	i, err := strconv.ParseInt(text, 10, 64)
	if err == nil {
		*d = Duration(time.Duration(i) * time.Second)
		return nil
	}
	f, err := strconv.ParseFloat(text, 64)
	*d = Duration(time.Duration(f) * time.Second)
	return err
}
