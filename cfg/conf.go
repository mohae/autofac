package cfg

import "flag"

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
