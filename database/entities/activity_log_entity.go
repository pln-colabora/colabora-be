package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityLog struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PermohonanID   uuid.UUID  `gorm:"type:uuid;not null" json:"permohonan_id"`
	Permohonan     Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ActivityNumber *int16     `gorm:"type:smallint" json:"activity_number"`
	Actor          uuid.UUID  `gorm:"type:uuid;not null" json:"actor"`
	ActorUser      User       `gorm:"foreignKey:Actor;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Action         string     `gorm:"type:varchar(100);not null" json:"action"`
	Detail         *string    `gorm:"type:text" json:"detail"`

	WorkflowNode *string `gorm:"type:varchar(50)" json:"workflow_node"`
	Timestamp
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}

func (l *ActivityLog) BeforeCreate(tx *gorm.DB) (err error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
