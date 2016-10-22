// package output exists for stringer generation because having it in
// autofactory results in go stringer errors.  This is probably a PEBKAC on my
// part but here it is.
package output

//go:generate stringer -type=OutputType
// OutputType is the type of destination the collected data is written to.
type OutputType int

const (
	Unsuported OutputType = iota
	File
	InfluxDB
)
