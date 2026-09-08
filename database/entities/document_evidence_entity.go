package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DocumentEvidence classifies a stored file as evidence for one workflow node.
// The migration adds composite foreign keys to enforce same-request attachment.
type DocumentEvidence struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	DocumentID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_document_node" json:"document_id"`
	PermohonanID uuid.UUID `gorm:"type:uuid;not null;index" json:"permohonan_id"`
	WorkflowNode string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_document_node" json:"workflow_node"`
	AttachedBy   uuid.UUID `gorm:"type:uuid;not null" json:"attached_by"`
	Timestamp
}

func (DocumentEvidence) TableName() string { return "document_evidence" }
func (e *DocumentEvidence) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
