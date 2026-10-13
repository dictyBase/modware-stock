package arangodb

import (
	fperrors "github.com/IBM/fp-go/v2/errors"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
)

// scanSearchRows drains one search cursor into the projection rows of a
// search feature. The go-driver cursor is a pull iterator, so the raw
// callback keeps the scan loop and the defer close, and msg names the
// feature in the error of a row read fault. The rows of an empty cursor
// stay a non-nil empty slice, because the callers report an empty list
// and not a missing one.
//
// fp-go:allow-trycatch-iferr the go-driver cursor reports a read
// error only inside the scan loop
func scanSearchRows[T any](
	msg string,
) func(dbRows) IOE.IOEither[error, []T] {
	return func(rows dbRows) IOE.IOEither[error, []T] {
		return F.Pipe1(
			IOE.TryCatchError(
				func() ([]T, error) {
					defer func() { _ = rows.Close() }()
					matches := []T{}
					for rows.Scan() {
						var row T
						if err := rows.Read(&row); err != nil {
							return nil, err
						}
						matches = append(matches, row)
					}
					return matches, nil
				},
			),
			IOE.MapLeft[[]T](fperrors.OnError(msg)),
		)
	}
}
