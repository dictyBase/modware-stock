package service

import (
	"context"
	"fmt"

	E "github.com/IBM/fp-go/either"
	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/collection"
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

// CreatePlasmid handles the creation of a new plasmid
func (s *StockService) CreatePlasmid(
	ctx context.Context,
	r *stock.NewPlasmid,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	stockDoc, err := s.repo.AddPlasmid(r)
	if err != nil {
		return plasmid, aphgrpc.HandleInsertError(ctx, err)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// GetPlasmid handles getting a plasmid by its ID using IOEither composition
func (s *StockService) GetPlasmid(
	ctx context.Context,
	req *stock.StockId,
) (*stock.Plasmid, error) {
	result := F.Pipe5(
		IOE.Of[error](getPlasmidContext{
			ctx:     ctx,
			request: req,
			repo:    s.repo,
		}),
		IOE.Bind(setValidatedRequest, validatePlasmidRequest),
		IOE.Bind(setStockDocument, retrievePlasmidFromRepository),
		IOE.Let[error](setPlasmidData, transformToPlasmidData),
		IOE.Map[error](extractPlasmidResponse),
		toServiceResult(ctx),
	)
	return result.F1, result.F2
}

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

// Curried setters for building context
var (
	// setValidatedRequest sets validated request ID in context
	setValidatedRequest = F.Curry2(
		func(validID string, pctx getPlasmidContext) withValidatedRequest {
			return withValidatedRequest{
				getPlasmidContext: pctx,
				validatedID:       validID,
			}
		},
	)

	// setStockDocument sets stock document in context
	setStockDocument = F.Curry2(
		func(doc *model.StockDoc, pctx withValidatedRequest) withStockDocument {
			return withStockDocument{
				withValidatedRequest: pctx,
				stockDoc:             doc,
			}
		},
	)

	// setPlasmidData sets plasmid data in context
	setPlasmidData = F.Curry2(
		func(data *stock.Plasmid_Data, pctx withStockDocument) withPlasmidData {
			return withPlasmidData{
				withStockDocument: pctx,
				plasmidData:       data,
			}
		},
	)

	// extractPlasmidResponse extracts plasmid response from enriched context
	extractPlasmidResponse = func(pctx withPlasmidData) *stock.Plasmid {
		return &stock.Plasmid{Data: pctx.plasmidData}
	}
)

// validatePlasmidRequest validates the stock ID request
func validatePlasmidRequest(
	pctx getPlasmidContext,
) IOE.IOEither[error, string] {
	return func() E.Either[error, string] {
		if err := pctx.request.Validate(); err != nil {
			return E.Left[string](
				fmt.Errorf("invalid request parameters: %w", err),
			)
		}
		return E.Right[error](pctx.request.Id)
	}
}

// retrievePlasmidFromRepository retrieves plasmid from repository
func retrievePlasmidFromRepository(
	pctx withValidatedRequest,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		pctx.repo.GetPlasmid(pctx.validatedID),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError(
				fmt.Sprintf("failed to retrieve plasmid %s", pctx.validatedID),
			),
		),
	)
}

// transformToPlasmidData transforms stock document to plasmid data using point-free composition
var transformToPlasmidData = F.Flow2(
	func(pctx withStockDocument) *model.StockDoc { return pctx.stockDoc },
	makePlasmidData,
)

// toServiceResult converts IOEither result to service response tuple with error handling
func toServiceResult(ctx context.Context) PlasmidConverter {
	return func(ioe PlasmidIO) PlasmidResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleGetError(ctx, e),
					)
				},
				func(p *stock.Plasmid) PlasmidResult {
					return T.MakeTuple2[*stock.Plasmid, error](
						p,
						nil,
					)
				},
			),
		)
	}
}

// LoadPlasmid loads plasmids with existing IDs into the database
func (s *StockService) LoadPlasmid(
	ctx context.Context,
	r *stock.ExistingPlasmid,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	id := r.Data.Id
	stockDoc, err := s.repo.LoadPlasmid(id, r)
	if err != nil {
		return plasmid, aphgrpc.HandleInsertError(ctx, err)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	// include ontology property if available
	if stockDoc.PlasmidProperties != nil {
		plasmid.Data.Attributes.DictyPlasmidProperty = stockDoc.PlasmidProperties.DictyPlasmidProperty
	}
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// UpdatePlasmid handles updating an existing plasmid
func (s *StockService) UpdatePlasmid(
	ctx context.Context,
	r *stock.PlasmidUpdate,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	stockDoc, err := s.repo.EditPlasmid(r)
	if err != nil {
		return plasmid, aphgrpc.HandleUpdateError(ctx, err)
	}
	if stockDoc.NotFound {
		return plasmid,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", stockDoc.ID),
			)
	}
	// Fetch the complete plasmid record to get all fields including ontology
	fullPlasmid, err := toTuple(s.repo.GetPlasmid(r.Data.Id))
	if err != nil {
		return plasmid, aphgrpc.HandleGetError(ctx, err)
	}
	plasmid.Data = makePlasmidData(fullPlasmid)
	err = s.publisher.PublishPlasmid(s.Topics["stockUpdate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(
	ctx context.Context,
	param *stock.StockParameters,
) (*stock.PlasmidCollection, error) {
	limit := limitVal(param.Limit)
	plasmidCollection := &stock.PlasmidCollection{
		Meta: &stock.Meta{Limit: limit},
	}
	stockDocs, err := stockModelList(&modelListParams{
		ctx:         ctx,
		stockParams: param,
		limit:       limit,
		fn:          s.repo.ListPlasmids,
	})
	if err != nil {
		return plasmidCollection, err
	}
	pdata := plasmidModelToCollectionSlice(stockDocs)
	if len(pdata) < int(limit)-2 { // fewer results than limit
		plasmidCollection.Data = pdata
		plasmidCollection.Meta.Total = int64(len(pdata))
		return plasmidCollection, nil
	}
	plasmidCollection.Data = pdata[:len(pdata)-1]
	plasmidCollection.Meta.NextCursor = genNextCursorVal(
		pdata[len(pdata)-1].Attributes.CreatedAt,
	)
	plasmidCollection.Meta.Total = int64(len(pdata))
	return plasmidCollection, nil
}

func makePlasmidData(m *model.StockDoc) *stock.Plasmid_Data {
	return &stock.Plasmid_Data{
		Type:       "plasmid",
		Id:         m.Key,
		Attributes: makePlasmidAttr(m),
	}
}

func plasmidModelToCollectionSlice(
	mc []*model.StockDoc,
) []*stock.PlasmidCollection_Data {
	return collection.Map(
		mc,
		func(m *model.StockDoc) *stock.PlasmidCollection_Data {
			return &stock.PlasmidCollection_Data{
				Type:       "plasmid",
				Id:         m.Key,
				Attributes: makePlasmidAttr(m),
			}
		},
	)
}

func makePlasmidAttr(m *model.StockDoc) *stock.PlasmidAttributes {
	attr := &stock.PlasmidAttributes{
		CreatedAt:       aphgrpc.TimestampProto(m.CreatedAt),
		UpdatedAt:       aphgrpc.TimestampProto(m.UpdatedAt),
		CreatedBy:       m.CreatedBy,
		UpdatedBy:       m.UpdatedBy,
		Summary:         m.Summary,
		EditableSummary: m.EditableSummary,
		Depositor:       m.Depositor,
		Genes:           m.Genes,
		Dbxrefs:         m.Dbxrefs,
		Publications:    m.Publications,
	}

	if m.PlasmidProperties != nil {
		attr.ImageMap = m.PlasmidProperties.ImageMap
		attr.Sequence = m.PlasmidProperties.Sequence
		attr.Name = m.PlasmidProperties.Name
		attr.DictyPlasmidProperty = m.PlasmidProperties.DictyPlasmidProperty
	}

	return attr
}

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
