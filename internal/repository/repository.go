package repository

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// StockRepository is an interface for accessing stock information
type StockRepository interface {
	GetStock(id string) (*model.StockDoc, error)
	AddStrain(ns *stock.NewStock) (*model.StockDoc, error)
	AddPlasmid(ns *stock.NewStock) (*model.StockDoc, error)
	EditStock(us *stock.StockUpdate) (*model.StockDoc, error)
	ListStocks(cursor int64, limit int64, filter string) ([]*model.StockDoc, error)
	ListStrains(cursor int64, limit int64) ([]*model.StockDoc, error)
	ListPlasmids(cursor int64, limit int64) ([]*model.StockDoc, error)
	RemoveStock(id string) error
	ClearStocks() error
}
