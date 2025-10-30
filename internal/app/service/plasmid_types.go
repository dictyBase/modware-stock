package service

import (
	"context"

	E "github.com/IBM/fp-go/either"
	IOE "github.com/IBM/fp-go/ioeither"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// Level 1: Basic Building Blocks - Core result types
type (
	// PlasmidResult represents a plasmid with potential error
	PlasmidResult = T.Tuple2[*stock.Plasmid, error]

	// PlasmidEither represents a computation that may succeed with plasmid or fail
	PlasmidEither = E.Either[error, *stock.Plasmid]

	// PlasmidIO represents an IO computation for plasmid retrieval
	PlasmidIO = IOE.IOEither[error, *stock.Plasmid]
)

// Level 2: Transformers and Converters - Function type aliases for transformations
type (
	// PlasmidConverter converts IOEither to Go's tuple result
	PlasmidConverter = func(PlasmidIO) PlasmidResult

	// PlasmidIOExecutor executes an IOEither to get Either
	PlasmidIOExecutor = func(PlasmidIO) PlasmidEither

	// PlasmidResultFolder folds Either into tuple result
	PlasmidResultFolder = func(PlasmidEither) PlasmidResult
)

// Level 3: Context-Aware Types - Context-parameterized transformations
type (
	// ContextualConverter is a converter factory that needs context
	ContextualConverter = func(context.Context) PlasmidConverter

	// ContextualErrorHandler wraps errors with context
	ContextualErrorHandler = func(context.Context) func(error) error
)

// StockRepository is a type alias for the repository interface
type StockRepository = repository.StockRepository

// getPlasmidContext represents the initial context for plasmid retrieval
type getPlasmidContext struct {
	ctx     context.Context
	request *stock.StockId
	repo    StockRepository
}

// withValidatedRequest adds validated request to context
type withValidatedRequest struct {
	getPlasmidContext
	validatedID string
}

// withStockDocument adds stock document to context
type withStockDocument struct {
	withValidatedRequest
	stockDoc *model.StockDoc
}

// withPlasmidData adds plasmid data to context
type withPlasmidData struct {
	withStockDocument
	plasmidData *stock.Plasmid_Data
}

// ListPlasmids workflow context types

// listPlasmidsContext represents the initial context for listing plasmids
type listPlasmidsContext struct {
	ctx   context.Context
	param *stock.StockParameters
	limit int64
	repo  StockRepository
}

// withValidatedFilter adds validated filter to context
type withValidatedFilter struct {
	listPlasmidsContext
	validatedFilter string
}

// withStockDocList adds stock document list to context
type withStockDocList struct {
	withValidatedFilter
	stockDocs []*model.StockDoc
}

// withPlasmidCollectionData adds plasmid collection data to context
type withPlasmidCollectionData struct {
	withStockDocList
	collectionData []*stock.PlasmidCollection_Data
}

// withNextCursor adds next cursor to context
type withNextCursor struct {
	withPlasmidCollectionData
	nextCursor int64
}

// Result types for ListPlasmids
type (
	// PlasmidCollectionResult represents a plasmid collection with potential error
	PlasmidCollectionResult = T.Tuple2[*stock.PlasmidCollection, error]

	// PlasmidCollectionEither represents computation that may succeed with collection or fail
	PlasmidCollectionEither = E.Either[error, *stock.PlasmidCollection]

	// PlasmidCollectionIO represents an IO computation for plasmid collection retrieval
	PlasmidCollectionIO = IOE.IOEither[error, *stock.PlasmidCollection]

	// PlasmidCollectionConverter converts IOEither to Go's tuple result
	PlasmidCollectionConverter = func(PlasmidCollectionIO) PlasmidCollectionResult
)
