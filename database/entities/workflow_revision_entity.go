package entities

import (
	"github.com/google/uuid"
	"time"
)

// ActivityRevision preserves the previous input and file selection without
// changing completion facts or the physical files shared with other nodes.
type ActivityRevision struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PermohonanID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_activity_revision" json:"-"`
	WorkflowNode string    `gorm:"not null;uniqueIndex:idx_activity_revision" json:"workflow_node"`
	Revision     int       `gorm:"not null;uniqueIndex:idx_activity_revision" json:"revision"`
	Payload      string    `gorm:"type:jsonb;not null" json:"-"`
	DocumentIDs  string    `gorm:"type:jsonb;not null" json:"-"`
	RecordedBy   uuid.UUID `gorm:"type:uuid;not null" json:"recorded_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type PDKBSchedule struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ScheduleDate time.Time  `gorm:"type:date;not null;uniqueIndex:idx_pdkb_schedule_date" json:"-"`
	Title        string     `gorm:"type:varchar(200);not null" json:"title"`
	PermohonanID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"permohonan_id"`
	CreatedBy    uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy    uuid.UUID  `gorm:"type:uuid;not null" json:"updated_by"`
	Timestamp
}
