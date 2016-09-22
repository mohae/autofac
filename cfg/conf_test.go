package cfg

import (
	"flag"
	"os"
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

func TestResolveEnvVars(t *testing.T) {
	tests := []struct {
		envs         map[string]string
		path         string
		resolvedPath string
		err          string
	}{
		{map[string]string{}, "$ZYXZ_MARGOS_TEST", "", "$ZYXZ_MARGOS_TEST: not set"},
		{map[string]string{"ZYXZ_MARGOS_TEST1": "path"}, "$ZYXZ_MARGOS_TEST1/dir", "path/dir", ""},
		{map[string]string{"ZYXZ_MARGOS_TEST1": "mydir", "ZYXZ_MARGOS_TEST2": "subdir"}, "$ZYXZ_MARGOS_TEST1/$ZYXZ_MARGOS_TEST2/dir", "mydir/subdir/dir", ""},
		{map[string]string{"ZYXZ_MARGOS_TEST1": "mydir", "ZYXZ_MARGOS_TEST2": "subdir"}, "$ZYXZ_MARGOS_TEST1/to/a/$ZYXZ_MARGOS_TEST2", "mydir/to/a/subdir", ""},
		{map[string]string{"ZYXZ_MARGOS_TEST1": "mydir", "ZYXZ_MARGOS_TEST2": "subdir"}, "$ZYXZ_MARGOS_TEST1/$ZYXZ_MARGOS_TEST2/$ZYXZ_MARGOS_TEST3/dir", "", "$ZYXZ_MARGOS_TEST3: not set"},
		{map[string]string{"ZYXZ_MARGOS_TEST1": "foo", "ZYXZ_MARGOS_TEST2": "bar", "ZYXZ_MARGOS_TEST3": "baz"}, "$ZYXZ_MARGOS_TEST1/$ZYXZ_MARGOS_TEST2/biz/$ZYXZ_MARGOS_TEST3/dir", "foo/bar/biz/baz/dir", ""},
	}
	var path string
	var err error
	for i, test := range tests {
		for k, v := range test.envs {
			err = os.Setenv(k, v)
			if err != nil {
				t.Errorf("%d: unexpected err: %s", i, err)
				goto unset
			}
		}

		path, err = ResolveEnvVars(test.path)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("%d: got %s, want %s", i, err, test.err)
			}
		} else {
			if test.err != "" {
				t.Errorf("%d: wanted %s; got no error", i, test.err)
			} else {
				if path != test.resolvedPath {
					t.Errorf("%d: got %q; want %q", i, path, test.path)
				}
			}
		}
	unset:
		for k := range test.envs {
			os.Unsetenv(k) // ignoring the returned error
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
