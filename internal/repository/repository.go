package repository

import (
	"io"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/model"
)

// StockRepository is an interface for managing stock information
type StockRepository interface {
	GetStrain(id string) (*model.StockDoc, error)
	GetPlasmid(id string) (*model.StockDoc, error)
	AddStrain(ns *stock.NewStrain) (*model.StockDoc, error)
	AddPlasmid(ns *stock.NewPlasmid) (*model.StockDoc, error)
	EditStrain(us *stock.StrainUpdate) (*model.StockDoc, error)
	EditPlasmid(us *stock.PlasmidUpdate) (*model.StockDoc, error)
	ListStrains(s *stock.StockParameters) ([]*model.StockDoc, error)
	ListStrainsByIds(s *stock.StockIdList) ([]*model.StockDoc, error)
	ListPlasmids(s *stock.StockParameters) ([]*model.StockDoc, error)
	LoadStrain(id string, es *stock.ExistingStrain) (*model.StockDoc, error)
	LoadPlasmid(id string, ep *stock.ExistingPlasmid) (*model.StockDoc, error)
	RemoveStock(id string) error
	Dbh() *manager.Database
	LoadOboJson(r io.Reader) (*storage.UploadInformation, error)
}
