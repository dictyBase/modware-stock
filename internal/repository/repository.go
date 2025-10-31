package repository

import (
	"io"

	IOE "github.com/IBM/fp-go/ioeither"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/model"
)

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
	Dbh() *manager.Database
	LoadOboJSON(r io.Reader) (*storage.UploadInformation, error)
}
