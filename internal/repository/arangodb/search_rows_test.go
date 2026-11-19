package arangodb

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

func TestAsDeadlineIfKilled(t *testing.T) {
	killed := driver.ArangoError{
		HasError:     true,
		Code:         500,
		ErrorNum:     driver.ErrQueryKilled,
		ErrorMessage: "AQL: query killed (while executing)",
	}
	other := driver.ArangoError{HasError: true, Code: 400, ErrorNum: 1501}
	require.ErrorIs(t, asDeadlineIfKilled(killed), context.DeadlineExceeded)
	var kept driver.ArangoError
	require.ErrorAs(t, asDeadlineIfKilled(killed), &kept,
		"expect the original Arango error to stay in the chain")
	require.Equal(t, driver.ErrQueryKilled, kept.ErrorNum)
	require.NotErrorIs(t, asDeadlineIfKilled(other), context.DeadlineExceeded)
	require.NotErrorIs(t, asDeadlineIfKilled(context.Canceled), context.DeadlineExceeded)
}

func TestSearchRowsKilledByMaxRuntimeIsDeadline(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ar, ok := repo.(*arangorepository)
	assert.True(ok, "expect the arangodb repository")
	ctx := driver.WithQueryMaxRuntime(context.Background(), 0.2)
	rows, err := ar.searchRows(ctx, "RETURN SLEEP(3)", nil)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Scan() {
			var v any
			err = rows.Read(&v)
		}
	}
	assert.ErrorIs(err, context.DeadlineExceeded, "expect a killed query to report a deadline")
}
