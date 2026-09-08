package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Document stores one private file, optionally bound to one permohonan.
// Evidence associations let it support multiple workflow nodes within that request.
type Document struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Type         string     `gorm:"type:varchar(50);not null" json:"type"`
	FilePath     string     `gorm:"type:varchar(500);not null" json:"-"`
	UploadedBy   uuid.UUID  `gorm:"type:uuid;not null" json:"uploaded_by"`
	Uploader     User       `gorm:"foreignKey:UploadedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PermohonanID *uuid.UUID `gorm:"type:uuid;index" json:"permohonan_id"`
	Permohonan   Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Evidence []DocumentEvidence `gorm:"foreignKey:DocumentID" json:"evidence"`
	Timestamp
}

func (Document) TableName() string { return "documents" }

func (d *Document) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
