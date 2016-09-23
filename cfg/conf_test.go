package cfg

import (
	"flag"
	"testing"
)

func TestConf(t *testing.T) {

	tests := []struct {
		Args          []string
		MissingName   []string
		ExpectedName  []string
		ExpectedValue []string
	}{
		{nil, nil, nil, nil},
		{[]string{"-foo", "F"}, []string{"bar", "biz", "baz"}, []string{"foo"}, []string{"F"}},
		{[]string{"-bar", "B"}, []string{"foo", "biz", "baz"}, []string{"bar"}, []string{"B"}},
		{[]string{"-baz", "Z", "-biz", "42"}, []string{"foo", "bar"}, []string{"baz", "biz"}, []string{"Z", "42"}},
		{[]string{"-foo", "F", "-bar", "B", "-baz", "Z", "-biz", "42"}, nil, []string{"foo", "bar", "baz", "biz"}, []string{"F", "B", "Z", "42"}},
	}
	for i, test := range tests {
		var conf Conf
		f := NewFlagSet()
		f.Parse(test.Args)
		f.Visit(conf.Visited)
		for _, arg := range test.MissingName {
			v := conf.Flag(arg)
			if v != nil {
				t.Errorf("%d: expected %s to not have been visited; it was: %v", i, arg, v)
			}
		}
		args := conf.Args()
		if len(args) != len(test.ExpectedName) {
			t.Errorf("%d: %d flags were visited; expected %d", i, len(args), len(test.ExpectedName))
			continue
		}
		for j, arg := range test.ExpectedName {
			v := conf.Flag(arg)
			if v == nil {
				t.Errorf("%d: expected %s to have been visited; it wasn't", i, arg)
				continue
			}
			if v.Value.String() != test.ExpectedValue[j] {
				t.Errorf("%d:%s: got %s want %s", i, arg, v.Value.String(), test.ExpectedValue)
			}
		}
	}
}

func NewFlagSet() *flag.FlagSet {
	f := flag.NewFlagSet("test", flag.ExitOnError)
	f.String("foo", "Foo", "string flag")
	f.String("bar", "Bar", "string flag")
	f.String("baz", "Baz", "string flag")
	f.String("biz", "Biz", "string flag")
	return f
}
