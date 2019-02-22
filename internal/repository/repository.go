package repository

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// StockRepository is an interface for accessing stock information
type StockRepository interface {
	GetStrain(id string) (*model.StockDoc, error)
	GetPlasmid(id string) (*model.StockDoc, error)
	AddStrain(ns *stock.NewStock) (*model.StockDoc, error)
	AddPlasmid(ns *stock.NewStock) (*model.StockDoc, error)
	EditPlasmid(us *stock.StockUpdate) (*model.StockDoc, error)
	EditStrain(us *stock.StockUpdate) (*model.StockDoc, error)
	ListStrains(s *stock.StockParameters) ([]*model.StockDoc, error)
	ListPlasmids(s *stock.StockParameters) ([]*model.StockDoc, error)
	RemoveStock(id string) error
	LoadStock(id string, es *stock.ExistingStock) (*model.StockDoc, error)
	ClearStocks() error
}
