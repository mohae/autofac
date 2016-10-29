// package output exists for stringer generation because having it in
// autofactory results in go stringer errors.  This is probably a PEBKAC on my
// part but here it is.
package output

import "strings"

//go:generate stringer -type=Type
// OutputType is the type of destination the collected data is written to.
type Type int

const (
	Unsupported Type = iota
	File
	InfluxDB
)

// TypeFromString returns the Type for a given string.  All input strings are
// normalized to lowercase; unmatched strings return Unsupported.
func TypeFromString(s string) Type {
	s = strings.ToLower(s)
	switch s {
	case "file":
		return File
	case "influxdb", "influx":
		return InfluxDB
	default:
		return Unsupported
	}
}
