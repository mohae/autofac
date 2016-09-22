package cfg

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Conf is used to hold flag arguments passed on start
type Conf struct {
	args []*flag.Flag // all flags that were visited
}

// Visited builds a list of the names of the flags that were passed as args.
func (c *Conf) Visited(a *flag.Flag) {
	c.args = append(c.args, a)
}

// Args returns all of the args that were passed.
func (c *Conf) Args() []*flag.Flag {
	return c.args
}

// Flag returns the requested flag, if it was set, or nil.
func (c *Conf) Flag(s string) *flag.Flag {
	for _, v := range c.args {
		if s == v.Name {
			return v
		}
	}
	return nil
}

// ResolveEnvVars takes a string and replaces any env vars found in the string
// with that env vars values.  The string is split on the OS's path separator;
// an env_var starts with a '$'.   If no env vars are present; the original
// string will be returned.  If an error occurs while getting an env variable
// value, it is returned.
func ResolveEnvVars(s string) (string, error) {
	parts := strings.Split(s, string(filepath.Separator))
	for i, v := range parts {
		if !strings.HasPrefix(v, "$") {
			continue
		}
		val := os.Getenv(strings.TrimPrefix(v, "$"))
		if val == "" {
			return "", fmt.Errorf("%s: not set", v)
		}
		parts[i] = val
	}
	return filepath.Join(parts...), nil
}
