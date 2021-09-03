package collection

import (
	"github.com/urfave/cli"
)

// MapString applies the given function to each element of string a, returning string slice of
// results
func MapString(a []string, fn func(string) string) []string {
	if len(a) == 0 {
		return a
	}
	sl := make([]string, len(a))
	for i, v := range a {
		sl[i] = fn(v)
	}
	return sl
}

// FilterFlags returns a new slice of cli Flags that
// passes the test implemented by the given functions
func FilterFlags(af []cli.Flag, fn func(cli.Flag) bool) []cli.Flag {
	if len(af) == 0 {
		return f
	}
	var nf []cli.Flag
	for _, f := range af {
		if !fn(f) {
			continue
		}
		nf = append(nf, f)
	}
	return nf
}
