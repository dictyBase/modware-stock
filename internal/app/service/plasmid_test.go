package service

import (
	"context"
	"testing"
)

func TestCreatePlasmid(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	t.Run("ValidPlasmid", func(_ *testing.T) {
		testCreateValidPlasmid(params)
	})

	t.Run("WithDefaultProperty", func(_ *testing.T) {
		testCreatePlasmidWithDefaultProperty(params)
	})

	t.Run("WithCustomProperty", func(_ *testing.T) {
		testCreatePlasmidWithCustomProperty(params)
	})

	t.Run("MinimalFields", func(_ *testing.T) {
		testCreatePlasmidMinimalFields(params)
	})

	t.Run("MissingRequiredFields", func(_ *testing.T) {
		testCreatePlasmidMissingRequiredFields(params)
	})

	t.Run("InvalidType", func(_ *testing.T) {
		testCreatePlasmidInvalidType(params)
	})

	t.Run("PublisherSuccess", func(_ *testing.T) {
		testCreatePlasmidPublisherSuccess(params)
	})

	t.Run("TimestampsSet", func(_ *testing.T) {
		testCreatePlasmidTimestampsSet(params)
	})
}

func TestGetPlasmid(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	t.Run("ExistingPlasmid", func(_ *testing.T) {
		testGetExistingPlasmid(params)
	})

	t.Run("NonExistent", func(_ *testing.T) {
		testGetNonExistentPlasmid(params)
	})

	t.Run("EmptyID", func(_ *testing.T) {
		testGetPlasmidWithEmptyID(params)
	})

	t.Run("InvalidID", func(_ *testing.T) {
		testGetPlasmidWithInvalidID(params)
	})
}

func TestLoadPlasmid(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	t.Run("Valid", func(_ *testing.T) {
		testLoadValidPlasmid(params)
	})

	t.Run("WithDefaultProperty", func(_ *testing.T) {
		testLoadPlasmidWithDefaultProperty(params)
	})

	t.Run("WithCustomProperty", func(_ *testing.T) {
		testLoadPlasmidWithCustomProperty(params)
	})

	t.Run("MissingRequiredFields", func(_ *testing.T) {
		testLoadPlasmidMissingRequiredFields(params)
	})

	t.Run("OntologyIncluded", func(_ *testing.T) {
		testLoadPlasmidOntologyIncluded(params)
	})
}

func TestUpdatePlasmid(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	t.Run("Existing", func(_ *testing.T) {
		testUpdateExistingPlasmid(params)
	})

	t.Run("NonExistent", func(_ *testing.T) {
		testUpdateNonExistentPlasmid(params)
	})

	t.Run("EmptyID", func(_ *testing.T) {
		testUpdatePlasmidWithEmptyID(params)
	})

	t.Run("PartialUpdate", func(_ *testing.T) {
		testUpdatePlasmidPartialUpdate(params)
	})

	t.Run("OntologyUpdate", func(_ *testing.T) {
		testUpdatePlasmidOntologyUpdate(params)
	})

	t.Run("OntologyWithOther", func(_ *testing.T) {
		testUpdatePlasmidOntologyWithOtherFields(params)
	})

	t.Run("InvalidOntology", func(_ *testing.T) {
		testUpdatePlasmidInvalidOntology(params)
	})

	t.Run("OntologyPreservation", func(_ *testing.T) {
		testUpdatePlasmidOntologyPreservation(params)
	})
}

func TestListPlasmids(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	t.Run("DefaultParams", func(_ *testing.T) {
		testListPlasmidsDefault(params)
	})

	t.Run("WithLimit", func(_ *testing.T) {
		testListPlasmidsWithLimit(params)
	})

	t.Run("WithLimitNoFilter", func(_ *testing.T) {
		testListPlasmidsWithLimitNoFilter(params)
	})

	t.Run("WithCursor", func(_ *testing.T) {
		testListPlasmidsWithCursor(params)
	})

	t.Run("Empty", func(_ *testing.T) {
		testListPlasmidsEmpty(params)
	})

	t.Run("InvalidFilter", func(_ *testing.T) {
		testListPlasmidsInvalidFilter(params)
	})

	t.Run("ByTagExact", func(_ *testing.T) {
		testListPlasmidsByTagExact(params)
	})

	t.Run("ByTagPartialMatch", func(_ *testing.T) {
		testListPlasmidsByTagPartialMatch(params)
	})

	t.Run("ByTagWithLimit", func(_ *testing.T) {
		testListPlasmidsByTagWithLimit(params)
	})

	t.Run("ByTagWithCursor", func(_ *testing.T) {
		testListPlasmidsByTagWithCursor(params)
	})

	t.Run("ByTagEmpty", func(_ *testing.T) {
		testListPlasmidsByTagEmpty(params)
	})

	t.Run("ByTagCombined", func(_ *testing.T) {
		testListPlasmidsByTagCombined(params)
	})
}
