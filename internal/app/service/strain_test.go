package service

import (
	"context"
	"testing"
)

// createTestParams creates test parameters for subtests
func createTestParams(t *testing.T, ctx context.Context, client *testClient, assert *testAssertions) *testParams {
	t.Helper()
	return &testParams{
		t:      t,
		ctx:    ctx,
		client: client,
		assert: assert,
	}
}

func TestCreateStrain(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()
	params := createTestParams(t, ctx, client, assert)

	t.Run("ValidStrain", func(t *testing.T) {
		testCreateValidStrain(createTestParams(t, ctx, client, assert))
	})

	t.Run("StrainWithDefaultProperty", func(t *testing.T) {
		testCreateStrainWithDefaultProperty(createTestParams(t, ctx, client, assert))
	})

	t.Run("StrainWithCustomProperty", func(t *testing.T) {
		testCreateStrainWithCustomProperty(createTestParams(t, ctx, client, assert))
	})

	t.Run("StrainMinimalFields", func(t *testing.T) {
		testCreateStrainMinimalFields(createTestParams(t, ctx, client, assert))
	})

	t.Run("StrainMissingRequiredFields", func(_ *testing.T) {
		testCreateStrainMissingRequiredFields(params)
	})

	t.Run("StrainInvalidType", func(_ *testing.T) {
		testCreateStrainInvalidType(params)
	})

	t.Run("StrainPublisherSuccess", func(t *testing.T) {
		testCreateStrainPublisherSuccess(createTestParams(t, ctx, client, assert))
	})

	t.Run("StrainTimestampsSet", func(t *testing.T) {
		testCreateStrainTimestampsSet(createTestParams(t, ctx, client, assert))
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

	t.Run("ExistingStrain", func(t *testing.T) {
		testUpdateExistingStrain(createTestParams(t, ctx, client, assert))
	})

	t.Run("NonExistentStrain", func(t *testing.T) {
		testUpdateNonExistentStrain(createTestParams(t, ctx, client, assert))
	})

	t.Run("EmptyID", func(t *testing.T) {
		testUpdateStrainWithEmptyID(createTestParams(t, ctx, client, assert))
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		testUpdateStrainPartialUpdate(createTestParams(t, ctx, client, assert))
	})

	t.Run("OntologyUpdate", func(t *testing.T) {
		testUpdateStrainOntologyUpdate(createTestParams(t, ctx, client, assert))
	})

	t.Run("OntologyWithOtherFields", func(t *testing.T) {
		testUpdateStrainOntologyWithOtherFields(createTestParams(t, ctx, client, assert))
	})

	t.Run("InvalidOntology", func(t *testing.T) {
		testUpdateStrainInvalidOntology(createTestParams(t, ctx, client, assert))
	})

	t.Run("OntologyPreservation", func(t *testing.T) {
		testUpdateStrainOntologyPreservation(createTestParams(t, ctx, client, assert))
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
