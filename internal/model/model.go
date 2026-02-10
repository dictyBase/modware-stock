// Package model defines data structures for biological stock management in the database layer.
package model

import (
	"time"

	driver "github.com/arangodb/go-driver"
)

// UploadStatus represents the status of an upload operation.
type UploadStatus int

// Upload status constants
const (
	Created UploadStatus = iota // Created indicates a new record was created
	Updated                     // Updated indicates an existing record was updated
	Failed                      // Failed indicates the upload operation failed
)

// StockDoc is the data structure for biological stocks
type StockDoc struct {
	driver.DocumentMeta
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	CreatedBy         string             `json:"created_by"`
	UpdatedBy         string             `json:"updated_by"`
	StockID           string             `json:"stock_id"`
	Summary           string             `json:"summary,omitempty"`
	EditableSummary   string             `json:"editable_summary,omitempty"`
	Depositor         string             `json:"depositor,omitempty"`
	Genes             []string           `json:"genes,omitempty"`
	Dbxrefs           []string           `json:"dbxrefs,omitempty"`
	Publications      []string           `json:"publications,omitempty"`
	StrainProperties  *StrainProperties  `json:"strain_properties,omitempty"`
	PlasmidProperties *PlasmidProperties `json:"plasmid_properties,omitempty"`
	NotFound          bool
}

// StrainProperties is the data structure for strain properties
type StrainProperties struct {
	DictyStrainProperty string   `json:"dicty_strain_property"`
	Label               string   `json:"label"`
	Species             string   `json:"species"`
	Plasmid             string   `json:"plasmid,omitempty"`
	Parent              string   `json:"parent,omitempty"`
	Names               []string `json:"names,omitempty"`
}

// PlasmidProperties is the data structure for plasmid properties
type PlasmidProperties struct {
	ImageMap             string `json:"image_map,omitempty" validate:"omitempty,url"`
	Sequence             string `json:"sequence,omitempty" validate:"omitempty,min=1"`
	Name                 string `json:"name" validate:"required,min=1,max=200"`
	DictyPlasmidProperty string `json:"dicty_plasmid_property" validate:"required"`
}
