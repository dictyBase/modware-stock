package model

import (
	"time"

	driver "github.com/arangodb/go-driver"
)

// StockDoc is the data structure for stock orders
type StockDoc struct {
	driver.DocumentMeta
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedBy       string    `json:"updated_by"`
	StockID         string    `json:"stock_id"`
	Summary         string    `json:"summary"`
	EditableSummary string    `json:"editable_summary"`
	Depositor       string    `json:"depositor"`
	Genes           []string  `json:"genes"`
	Dbxrefs         []string  `json:"dbxrefs"`
	Publications    []string  `json:"publications"`
	SystematicName  string    `json:"systematic_name"`
	Descriptor      string    `json:"descriptor"`
	Species         string    `json:"species"`
	Plasmid         string    `json:"plasmid"`
	Parents         []string  `json:"parents"`
	Names           []string  `json:"names"`
	ImageMap        string    `json:"image_map"`
	Sequence        string    `json:"sequence"`
	Keywords        []string  `json:"keywords"`
	NotFound        bool
}
