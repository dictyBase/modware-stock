// Package repository defines interfaces for persisting and retrieving biological stock data.
package repository

import (
	"io"

	IOE "github.com/IBM/fp-go/ioeither"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/model"
)

// StockEntityFilter restricts a stock search to one kind of stock. The
// zero value covers both kinds. The values match the type attribute of
// the edge in the stock property graph.
type StockEntityFilter string

const (
	// EntityBoth covers strains and plasmids.
	EntityBoth StockEntityFilter = ""
	// EntityStrain covers strains only.
	EntityStrain StockEntityFilter = "strain"
	// EntityPlasmid covers plasmids only.
	EntityPlasmid StockEntityFilter = "plasmid"
)

// AutocompleteQuery holds the input of an autocomplete search. The
// repository normalizes Query and clamps Limit.
type AutocompleteQuery struct {
	// Query is the partial search text. The caller sends it untrimmed;
	// the repository trims it and lowercases it.
	Query string
	// Entity restricts the search to one kind of stock.
	Entity StockEntityFilter
	// Limit is the maximum number of suggestions. A value at or below 0
	// becomes 5. A value above 50 becomes 50.
	Limit int
}

// Suggestion is one autocomplete match.
type Suggestion struct {
	// ID is the stock_id of the matched stock, for example DBS0236126.
	ID string
	// Field is the field path that matched, for example label or genes.
	Field string
	// DisplayText is the text that the user interface shows.
	DisplayText string
	// Entity is the kind of the matched stock. It is never EntityBoth.
	Entity StockEntityFilter
	// Score ranks the match. A prefix match scores above a fuzzy match.
	Score float64
}

// StockRepository is an interface for managing stock information
type StockRepository interface {
	GetStrain(id string) (*model.StockDoc, error)
	GetPlasmid(id string) IOE.IOEither[error, *model.StockDoc]
	AddStrain(ns *stock.NewStrain) (*model.StockDoc, error)
	AddPlasmid(ns *stock.NewPlasmid) IOE.IOEither[error, *model.StockDoc]
	EditStrain(us *stock.StrainUpdate) (*model.StockDoc, error)
	EditPlasmid(us *stock.PlasmidUpdate) IOE.IOEither[error, *model.StockDoc]
	ListStrains(s *stock.StockParameters) ([]*model.StockDoc, error)
	ListStrainsByIDs(s *stock.StockIdList) ([]*model.StockDoc, error)
	ListPlasmids(s *stock.StockParameters) IOE.IOEither[error, []*model.StockDoc]
	LoadStrain(id string, es *stock.ExistingStrain) (*model.StockDoc, error)
	LoadPlasmid(id string, ep *stock.ExistingPlasmid) IOE.IOEither[error, *model.StockDoc]
	RemoveStock(id string) error
	// AutocompleteStock returns at most Limit short suggestions for the
	// normalized query of params.
	AutocompleteStock(params *AutocompleteQuery) ([]*Suggestion, error)
	Dbh() *manager.Database
	LoadOboJSON(r io.Reader) (*storage.UploadInformation, error)
}
