package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Document is upload-first: a file is uploaded standalone (PermohonanID/ActivityNumber nil)
// via POST /api/documents, then attached to exactly one permohonan+activity later by
// DocumentService.AttachToActivity — a plain nullable-FK update, not a many-to-many pivot,
// since a document is never attached to more than one permohonan/activity, ever.
type Document struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Type           string     `gorm:"type:varchar(50);not null" json:"type"`
	FilePath       string     `gorm:"type:varchar(500);not null" json:"-"`
	UploadedBy     uuid.UUID  `gorm:"type:uuid;not null" json:"uploaded_by"`
	Uploader       User       `gorm:"foreignKey:UploadedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PermohonanID   *uuid.UUID `gorm:"type:uuid;index" json:"permohonan_id"`
	Permohonan     Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ActivityNumber *int16     `gorm:"type:smallint;index" json:"activity_number"`

	Timestamp
}

func (Document) TableName() string { return "documents" }

func (d *Document) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
