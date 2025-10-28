package service

import (
	"context"
	"testing"
)

func TestCreateStrain(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}

	// Test cases for CreateStrain
	t.Run("ValidStrain", func(t *testing.T) {
		testCreateValidStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("StrainWithDefaultProperty", func(t *testing.T) {
		testCreateStrainWithDefaultProperty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("StrainWithCustomProperty", func(t *testing.T) {
		testCreateStrainWithCustomProperty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("StrainMinimalFields", func(t *testing.T) {
		testCreateStrainMinimalFields(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("StrainMissingRequiredFields", func(t *testing.T) {
		testCreateStrainMissingRequiredFields(params)
	})

	t.Run("StrainInvalidType", func(t *testing.T) {
		testCreateStrainInvalidType(params)
	})

	t.Run("StrainPublisherSuccess", func(t *testing.T) {
		testCreateStrainPublisherSuccess(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("StrainTimestampsSet", func(t *testing.T) {
		testCreateStrainTimestampsSet(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}

func TestGetStrain(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	// Test cases for GetStrain
	t.Run("ExistingStrain", func(t *testing.T) {
		testGetExistingStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("NonExistentStrain", func(t *testing.T) {
		testGetNonExistentStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("EmptyID", func(t *testing.T) {
		testGetStrainWithEmptyID(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("InvalidID", func(t *testing.T) {
		testGetStrainWithInvalidID(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}

func TestLoadStrain(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	// Test cases for LoadStrain
	t.Run("ValidStrain", func(t *testing.T) {
		testLoadValidStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("WithDefaultProperty", func(t *testing.T) {
		testLoadStrainWithDefaultProperty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("WithCustomProperty", func(t *testing.T) {
		testLoadStrainWithCustomProperty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		testLoadStrainMissingRequiredFields(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}

func TestUpdateStrain(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	// Test cases for UpdateStrain
	t.Run("ExistingStrain", func(t *testing.T) {
		testUpdateExistingStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("NonExistentStrain", func(t *testing.T) {
		testUpdateNonExistentStrain(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("EmptyID", func(t *testing.T) {
		testUpdateStrainWithEmptyID(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		testUpdateStrainPartialUpdate(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("OntologyUpdate", func(t *testing.T) {
		testUpdateStrainOntologyUpdate(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("OntologyWithOtherFields", func(t *testing.T) {
		testUpdateStrainOntologyWithOtherFields(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("InvalidOntology", func(t *testing.T) {
		testUpdateStrainInvalidOntology(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("OntologyPreservation", func(t *testing.T) {
		testUpdateStrainOntologyPreservation(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}

func TestListStrainsByIDs(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	// Test cases for ListStrainsByIDs
	t.Run("WithExisting", func(t *testing.T) {
		testListStrainsByIDsWithExisting(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("NonExistent", func(t *testing.T) {
		testListStrainsByIDsNonExistent(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("EmptyList", func(t *testing.T) {
		testListStrainsByIDsEmpty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("MixedIDs", func(t *testing.T) {
		testListStrainsByIDsMixed(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}

func TestListStrains(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	// Test cases for ListStrains
	t.Run("DefaultParameters", func(t *testing.T) {
		testListStrainsDefault(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("WithLimit", func(t *testing.T) {
		testListStrainsWithLimit(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("WithCursor", func(t *testing.T) {
		testListStrainsWithCursor(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})

	t.Run("Empty", func(t *testing.T) {
		testListStrainsEmpty(&testParams{
			t:      t,
			ctx:    ctx,
			client: client,
			assert: assert,
		})
	})
}
