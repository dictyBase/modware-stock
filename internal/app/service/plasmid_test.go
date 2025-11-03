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

	t.Run("ValidPlasmid", func(t *testing.T) {
		testCreateValidPlasmid(params)
	})

	t.Run("WithDefaultProperty", func(t *testing.T) {
		testCreatePlasmidWithDefaultProperty(params)
	})

	t.Run("WithCustomProperty", func(t *testing.T) {
		testCreatePlasmidWithCustomProperty(params)
	})

	t.Run("MinimalFields", func(t *testing.T) {
		testCreatePlasmidMinimalFields(params)
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCreatePlasmidMissingRequiredFields(params)
	})

	t.Run("InvalidType", func(t *testing.T) {
		testCreatePlasmidInvalidType(params)
	})

	t.Run("PublisherSuccess", func(t *testing.T) {
		testCreatePlasmidPublisherSuccess(params)
	})

	t.Run("TimestampsSet", func(t *testing.T) {
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

	t.Run("ExistingPlasmid", func(t *testing.T) {
		testGetExistingPlasmid(params)
	})

	t.Run("NonExistent", func(t *testing.T) {
		testGetNonExistentPlasmid(params)
	})

	t.Run("EmptyID", func(t *testing.T) {
		testGetPlasmidWithEmptyID(params)
	})

	t.Run("InvalidID", func(t *testing.T) {
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

	t.Run("Valid", func(t *testing.T) {
		testLoadValidPlasmid(params)
	})

	t.Run("WithDefaultProperty", func(t *testing.T) {
		testLoadPlasmidWithDefaultProperty(params)
	})

	t.Run("WithCustomProperty", func(t *testing.T) {
		testLoadPlasmidWithCustomProperty(params)
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		testLoadPlasmidMissingRequiredFields(params)
	})

	t.Run("OntologyIncluded", func(t *testing.T) {
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

	t.Run("Existing", func(t *testing.T) {
		testUpdateExistingPlasmid(params)
	})

	t.Run("NonExistent", func(t *testing.T) {
		testUpdateNonExistentPlasmid(params)
	})

	t.Run("EmptyID", func(t *testing.T) {
		testUpdatePlasmidWithEmptyID(params)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		testUpdatePlasmidPartialUpdate(params)
	})

	t.Run("OntologyUpdate", func(t *testing.T) {
		testUpdatePlasmidOntologyUpdate(params)
	})

	t.Run("OntologyWithOther", func(t *testing.T) {
		testUpdatePlasmidOntologyWithOtherFields(params)
	})

	t.Run("InvalidOntology", func(t *testing.T) {
		testUpdatePlasmidInvalidOntology(params)
	})

	t.Run("OntologyPreservation", func(t *testing.T) {
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

	t.Run("DefaultParams", func(t *testing.T) {
		testListPlasmidsDefault(params)
	})

	t.Run("WithLimit", func(t *testing.T) {
		testListPlasmidsWithLimit(params)
	})

	t.Run("WithCursor", func(t *testing.T) {
		testListPlasmidsWithCursor(params)
	})

	t.Run("Empty", func(t *testing.T) {
		testListPlasmidsEmpty(params)
	})

	t.Run("InvalidFilter", func(t *testing.T) {
		testListPlasmidsInvalidFilter(params)
	})
}
