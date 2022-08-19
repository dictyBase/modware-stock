package collection

import (
	"github.com/urfave/cli"
)

// Map applies the given function to each element of slice a, returning slice of
// results
func Map[T any, M any](a []T, fn func(T) M) []M {
	anySlice := make([]M, 0)
	for _, elem := range a {
		anySlice = append(anySlice, fn(elem))
	}

	return anySlice
}

// FilterFlags returns a new slice of cli Flags that
// passes the test implemented by the given functions
func FilterFlags(af []cli.Flag, fn func(cli.Flag) bool) []cli.Flag {
	if len(af) == 0 {
		return af
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
