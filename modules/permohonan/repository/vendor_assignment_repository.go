package repository

import (
	"context"
	"errors"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrVendorAlreadyAssigned = errors.New("vendor role already assigned")

type VendorAssignmentRepository interface {
	Assign(context.Context, *gorm.DB, entities.VendorAssignment) error
}
type vendorAssignmentRepository struct{}

func NewVendorAssignmentRepository() VendorAssignmentRepository { return &vendorAssignmentRepository{} }

// Caller holds the aggregate lock and owns the transaction.
func (*vendorAssignmentRepository) Assign(ctx context.Context, tx *gorm.DB, assignment entities.VendorAssignment) error {
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&assignment)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrVendorAlreadyAssigned
	}
	return tx.WithContext(ctx).Create(&entities.ActivityLog{
		PermohonanID: assignment.PermohonanID, Actor: assignment.AssignedBy, Action: "vendor_assigned",
	}).Error
}
