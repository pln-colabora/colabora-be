package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountDocument links a private uploaded document to the user account it
// was submitted for. It is intentionally separate from DocumentEvidence,
// which is reserved for workflow evidence belonging to a Permohonan.
type AccountDocument struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	DocumentID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"document_id"`
	DocumentType string    `gorm:"type:varchar(50);not null" json:"document_type"`

	User     User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Document Document `gorm:"foreignKey:DocumentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Timestamp
}

func (AccountDocument) TableName() string { return "account_documents" }

func (d *AccountDocument) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
