package sysinfo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/google/flatbuffers/go"
)

// MemDataTicker gets the current memeory data on a ticker and outputs the
// results as a flatbuffer serialized []byte.  The data is gathered using the
// free -k command.
func MemDataTicker(interval time.Duration, outCh chan []byte) {
	ticker := time.NewTicker(interval)
	defer close(outCh)
	defer ticker.Stop()
	var out bytes.Buffer
	// lnum is the line number being processed
	// fieldNum is the field being processed
	// pos is the current position in the line
	// ndx is the current index in the byte slice used to hold the value of the field being processed
	// i is the holder for Atoi results
	var lNum, fldNum, pos, ndx, i int
	var v byte
	// new builder is created outside the loop and reset at the end of each tick
	bldr := flatbuffers.NewBuilder(0)
	for {
		select {
		case <-ticker.C:
			cmd := exec.Command("free", "-k")
			cmd.Stdout = &out
			// get the current time in millisecond resolution; always as UTC
			t := time.Now().UTC().UnixNano()
			err := cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error getting memory stats: %s\n", err)
			}
			MemDataStart(bldr)
			MemDataAddTimestamp(bldr, t)
			// holds the current field value
			fld := make([]byte, 12)
			// process the output
			for {
				line, err := out.ReadBytes(nl)
				if err != nil {
					if err == io.EOF {
						break
					}
					fmt.Fprintf(os.Stderr, "error reading bytes from free command results: %s\n", err)
				}
				// if the first char is a space; skip
				if line[0] == 0x20 {
					continue
				}
				// skip the beginning text (go to :)
				for i, v = range line {
					if v == 0x3A {
						pos += i + 1
						break
					}
				}
				lNum++
				// process the fields
				for _, v = range line[pos:] {
					// if we are at a space, see if the
					if v == 0x20 || v == nl {
						if ndx == 0 {
							continue
						}
						i, _ = strconv.Atoi(string(fld[:ndx]))
						switch fldNum {
						case 0:
							MemDataAddRAMTotal(bldr, int64(i))
						case 1:
							MemDataAddRAMUsed(bldr, int64(i))
						case 2:
							MemDataAddRAMFree(bldr, int64(i))
						case 3:
							MemDataAddRAMShared(bldr, int64(i))
						case 4:
							MemDataAddRAMBuffers(bldr, int64(i))
						case 5:
							MemDataAddRAMCached(bldr, int64(i))
						case 6:
							MemDataAddCacheUsed(bldr, int64(i))
						case 7:
							MemDataAddCacheFree(bldr, int64(i))
						case 8:
							MemDataAddSwapTotal(bldr, int64(i))
						case 9:
							MemDataAddSwapUsed(bldr, int64(i))
						case 10:
							MemDataAddSwapFree(bldr, int64(i))
						}
						fldNum++
						ndx = 0
						continue
					}
					fld[ndx] = v
					ndx++
				}
				pos = 0
			}
			bldr.Finish(MemDataEnd(bldr))
			// get the bytes
			tmp := bldr.Bytes[bldr.Head():]
			// copy them (otherwise gets lost in reset)
			cpy := make([]byte, len(tmp))
			copy(cpy, tmp)
			// send the bytes to the enqueue func
			outCh <- cpy
			// reset foor next tick
			bldr.Reset()
			pos, lNum, ndx, fldNum = 0, 0, 0, 0
			out.Reset()
		}
	}
}

// UnmarshalMemDataToString takes a flatbuffers serialized []byte and returns
// the data as a formatted string.
func UnmarshalMemDataToString(p []byte) string {
	m := GetRootAsMemData(p, 0)
	return fmt.Sprintf("%d\n%d\t%d\t%d\t%d\t%d\t%d\n%d\t%d\n%d\t%d\t%d\n", m.Timestamp(), m.RAMTotal(), m.RAMUsed(), m.RAMFree(), m.RAMShared(), m.RAMBuffers(), m.RAMCached(), m.CacheUsed(), m.CacheFree(), m.SwapTotal(), m.SwapUsed(), m.SwapFree())
}
