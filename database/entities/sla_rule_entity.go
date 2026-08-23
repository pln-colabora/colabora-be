package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SLARule struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ActivityNumber int16     `gorm:"type:smallint;not null;uniqueIndex:idx_sla_rule_activity_jenis" json:"activity_number"`
	ActivityName   string    `gorm:"type:varchar(100);not null" json:"activity_name"`
	JenisSambungan string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_sla_rule_activity_jenis" json:"jenis_sambungan"`
	OffsetDays     int16     `gorm:"type:smallint;not null" json:"offset_days"`
	ReferencePoint string    `gorm:"type:varchar(1);not null" json:"reference_point"`

	Timestamp
}

func (s *SLARule) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
