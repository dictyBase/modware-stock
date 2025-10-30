package service

import (
	E "github.com/IBM/fp-go/either"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	T "github.com/IBM/fp-go/tuple"
)

// toTuple converts IOEither to Go tuple using functional Tuple2 approach
func toTuple[A any](ioe IOE.IOEither[error, A]) (A, error) {
	result := F.Pipe1(
		ioe(), // Execute IOEither to get Either
		E.Fold(
			func(e error) T.Tuple2[A, error] {
				var zero A
				return T.MakeTuple2(zero, e)
			},
			func(data A) T.Tuple2[A, error] {
				return T.MakeTuple2[A, error](data, nil)
			},
		),
	)
	return result.F1, result.F2
}
