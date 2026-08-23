package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermohonanActivity struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PermohonanID   uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_permohonan_activity_number" json:"permohonan_id"`
	Permohonan     Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ActivityNumber int16      `gorm:"type:smallint;not null;uniqueIndex:idx_permohonan_activity_number" json:"activity_number"`
	StageNumber    int16      `gorm:"type:smallint;not null" json:"stage_number"`
	Status         string     `gorm:"type:varchar(20);not null;default:'not_started'" json:"status"`
	SlaDeadline    time.Time  `gorm:"type:date;not null" json:"sla_deadline"`
	Payload        string     `gorm:"type:jsonb;default:'{}'" json:"payload"`
	CompletedBy    *uuid.UUID `gorm:"type:uuid" json:"completed_by"`
	CompletedAt    *time.Time `json:"completed_at"`

	Timestamp
}

func (PermohonanActivity) TableName() string {
	return "permohonan_activities"
}

func (a *PermohonanActivity) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
