package arangodb

import (
	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// LoadPlasmid will insert existing plasmid data into the database.
// It receives the already existing plasmid ID and the data to go with it.
func (ar *arangorepository) LoadPlasmid(
	id string,
	ep *stock.ExistingPlasmid,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe6(
		IOE.Of[error](ep),
		IOE.Chain(validateExistingPlasmidInput),
		IOE.Map[error](
			func(validated *stock.ExistingPlasmid) loadPlasmidParams {
				return loadPlasmidParams{id: id, plasmid: validated}
			},
		),
		IOE.Chain(ar.validateLoadPlasmidOntologyTerm),
		IOE.Map[error](ar.buildLoadPlasmidParams),
		IOE.Chain(ar.executeLoadPlasmidQuery),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to load plasmid"),
		),
	)
}

// EditPlasmid updates an existing plasmid
func (ar *arangorepository) EditPlasmid(
	us *stock.PlasmidUpdate,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe6(
		IOE.Of[error](us),
		IOE.Chain(validatePlasmidUpdateInput),
		IOE.Chain(ar.validateEditPlasmidStock),
		IOE.Chain(ar.updatePlasmidOntologyTerm),
		IOE.Map[error](ar.buildEditPlasmidParams),
		IOE.Chain(ar.executeEditPlasmidQuery),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to edit plasmid"),
		),
	)
}

// AddPlasmid creates a new plasmid stock
func (ar *arangorepository) AddPlasmid(
	ns *stock.NewPlasmid,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe5(
		IOE.Of[error](ns),
		IOE.Chain(validateNewPlasmidInput),
		IOE.Chain(ar.validateAddPlasmidOntologyTerm),
		IOE.Map[error](ar.buildAddPlasmidParams),
		IOE.Chain(ar.executeAddPlasmidQuery),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to add plasmid"),
		),
	)
}
