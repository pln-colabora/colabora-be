package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailNotification struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	DeduplicationKey string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	Recipient        string     `gorm:"type:varchar(255);not null;index" json:"recipient"`
	Subject          string     `gorm:"type:varchar(255);not null" json:"subject"`
	Body             string     `gorm:"type:text;not null" json:"-"`
	Status           string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Attempts         int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt    time.Time  `gorm:"not null;index" json:"next_attempt_at"`
	LockedUntil      *time.Time `json:"-"`
	SentAt           *time.Time `json:"sent_at"`
	LastError        *string    `gorm:"type:text" json:"-"`

	Timestamp
}

func (EmailNotification) TableName() string { return "email_notifications" }

func (n *EmailNotification) BeforeCreate(_ *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
