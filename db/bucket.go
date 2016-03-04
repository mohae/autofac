//go:generate stringer -type=Bucket
package db

import (
	"strings"
)

type Bucket int

const (
	Invalid Bucket = iota
	Client
	Role
	Group
	Cluster
	Datacenter
)

// formats is a slice of supported Formats
var Buckets = []Bucket{Invalid, Client, Role, Group, Cluster, Datacenter}

// BucketFromString returns the Bucket for a given string, or Invalid for
// anything that does not match.  All input strings are normalized to lower.
func BucketFromString(s string) Bucket {
	s = strings.ToLower(s)
	switch s {
	case "client":
		return Client
	case "role":
		return Role
	case "group":
		return Group
	case "cluster":
		return Cluster
	case "datacenter":
		return Datacenter
	default:
		return Invalid
	}
}
