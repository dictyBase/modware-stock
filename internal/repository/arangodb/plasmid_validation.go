package arangodb

import (
	"fmt"

	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	validator "github.com/go-playground/validator/v10"
)

// Package-level validator instance for input validation (thread-safe singleton)
var validate = validator.New()

// Validation wrapper structs for input validation

// newPlasmidValidation wraps NewPlasmid attributes for validation
type newPlasmidValidation struct {
	CreatedBy string   `validate:"required,email"`
	UpdatedBy string   `validate:"required,email"`
	Depositor string   `validate:"required,min=2"`
	Name      string   `validate:"omitempty,min=2,max=255"`
	Summary   string   `validate:"omitempty,min=10"`
	Genes     []string `validate:"omitempty,dive,required,min=1"`
}

// plasmidUpdateValidation wraps PlasmidUpdate attributes for validation
type plasmidUpdateValidation struct {
	UpdatedBy       string   `validate:"required,email"`
	Depositor       string   `validate:"omitempty,min=2"`
	Name            string   `validate:"omitempty,min=2,max=255"`
	Summary         string   `validate:"omitempty,min=10"`
	EditableSummary string   `validate:"omitempty,min=10"`
	Genes           []string `validate:"omitempty,dive,required,min=1"`
}

// existingPlasmidValidation wraps ExistingPlasmid attributes for validation
type existingPlasmidValidation struct {
	CreatedBy string   `validate:"required,email"`
	UpdatedBy string   `validate:"required,email"`
	Depositor string   `validate:"required,min=2"`
	Name      string   `validate:"omitempty,min=2,max=255"`
	Summary   string   `validate:"omitempty,min=10"`
	Genes     []string `validate:"omitempty,dive,required,min=1"`
}

// validateNewPlasmidInput validates input for AddPlasmid operation
func validateNewPlasmidInput(
	ns *stock.NewPlasmid,
) IOE.IOEither[error, *stock.NewPlasmid] {
	return IOE.TryCatchError(
		func() (*stock.NewPlasmid, error) {
			req := newPlasmidValidation{
				CreatedBy: ns.Data.Attributes.CreatedBy,
				UpdatedBy: ns.Data.Attributes.UpdatedBy,
				Depositor: ns.Data.Attributes.Depositor,
				Name:      ns.Data.Attributes.Name,
				Summary:   ns.Data.Attributes.Summary,
				Genes:     ns.Data.Attributes.Genes,
			}

			if err := validate.Struct(req); err != nil {
				return nil, fmt.Errorf(
					"validation failed for new plasmid: %w",
					err,
				)
			}

			return ns, nil
		},
	)
}

// validatePlasmidUpdateInput validates input for EditPlasmid operation
func validatePlasmidUpdateInput(
	us *stock.PlasmidUpdate,
) IOE.IOEither[error, *stock.PlasmidUpdate] {
	return IOE.TryCatchError(
		func() (*stock.PlasmidUpdate, error) {
			req := plasmidUpdateValidation{
				UpdatedBy:       us.Data.Attributes.UpdatedBy,
				Depositor:       us.Data.Attributes.Depositor,
				Name:            us.Data.Attributes.Name,
				Summary:         us.Data.Attributes.Summary,
				EditableSummary: us.Data.Attributes.EditableSummary,
				Genes:           us.Data.Attributes.Genes,
			}

			if err := validate.Struct(req); err != nil {
				return nil, fmt.Errorf(
					"validation failed for plasmid update: %w",
					err,
				)
			}

			return us, nil
		},
	)
}

// validateExistingPlasmidInput validates input for LoadPlasmid operation
func validateExistingPlasmidInput(
	ep *stock.ExistingPlasmid,
) IOE.IOEither[error, *stock.ExistingPlasmid] {
	return IOE.TryCatchError(
		func() (*stock.ExistingPlasmid, error) {
			req := existingPlasmidValidation{
				CreatedBy: ep.Data.Attributes.CreatedBy,
				UpdatedBy: ep.Data.Attributes.UpdatedBy,
				Depositor: ep.Data.Attributes.Depositor,
				Name:      ep.Data.Attributes.Name,
				Summary:   ep.Data.Attributes.Summary,
				Genes:     ep.Data.Attributes.Genes,
			}

			if err := validate.Struct(req); err != nil {
				return nil, fmt.Errorf(
					"validation failed for existing plasmid: %w",
					err,
				)
			}

			return ep, nil
		},
	)
}
