package entities

import "github.com/google/uuid"

// VendorAssignment grants one vendor account access to a request for its role.
// Assignments are immutable through the API until reassignment is specified.
type VendorAssignment struct {
	PermohonanID uuid.UUID  `gorm:"type:uuid;primaryKey" json:"permohonan_id"`
	VendorRole   string     `gorm:"type:varchar(50);primaryKey;check:vendor_assignment_role,vendor_role IN ('vendor-tiang','vendor-konstruksi','vendor-sr-app')" json:"vendor_role"`
	VendorID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"vendor_id"`
	AssignedBy   uuid.UUID  `gorm:"type:uuid;not null" json:"assigned_by"`
	Permohonan   Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnDelete:CASCADE" json:"-"`
	Vendor       User       `gorm:"foreignKey:VendorID;constraint:OnDelete:RESTRICT" json:"-"`
	Assigner     User       `gorm:"foreignKey:AssignedBy;constraint:OnDelete:RESTRICT" json:"-"`
	Timestamp
}
