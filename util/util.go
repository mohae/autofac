package util

import (
    crand "crypto/rand"
    "fmt"
    "math/big"
    "math/rand"
)

const max64 = 1 << 63 -1
const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var prng = rand.New(rand.NewSource(seed()))

func init() {
    seed()
}

// seed gets a random int64 using a CSPRNG.
func seed() int64 {
    bi := big.NewInt(max64)
    r, err := crand.Int(crand.Reader, bi)
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
        id[i] = alphanum[prng.Intn(len(alphanum))]
    }
    return string(id)
}

// returns a uint32 uptained from prng
func RandUint32() uint32 {
    return prng.Uint32()
}

func Uint32ToBytes(x uint32) [4]byte {
	var b [4]byte
	b[0] = byte(x>>24)
	b[1] = byte(x>>16)
	b[2] = byte(x>>8)
	b[3] = byte(x)
	return b
}

func Int64ToBytes(x int64) [8]byte {
	var b [8]byte
	b[0] = byte(x>>56)
	b[1] = byte(x>>48)
	b[2] = byte(x>>40)
	b[3] = byte(x>>32)
	b[4] = byte(x>>24)
	b[5] = byte(x>>16)
	b[6] = byte(x>>8)
	b[7] = byte(x)
	return b
}
