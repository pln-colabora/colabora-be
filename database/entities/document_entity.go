package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Document stores one private file, optionally bound to one permohonan.
// Evidence associations let it support multiple workflow nodes within that request.
type Document struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Type             string     `gorm:"type:varchar(50);not null" json:"type"`
	FilePath         string     `gorm:"type:varchar(500);not null" json:"-"`
	OriginalFilename string     `gorm:"type:varchar(255);not null" json:"original_filename"`
	MimeType         string     `gorm:"type:varchar(100);not null" json:"mime_type"`
	SizeBytes        int64      `gorm:"not null" json:"size_bytes"`
	ChecksumSHA256   string     `gorm:"type:char(64);not null" json:"checksum_sha256"`
	Source           string     `gorm:"type:varchar(20);not null;default:uploaded" json:"source"`
	Classification   string     `gorm:"type:varchar(20);not null;default:restricted" json:"classification"`
	ScanStatus       string     `gorm:"type:varchar(20);not null;default:not_scanned" json:"scan_status"`
	ScanCheckedAt    *time.Time `gorm:"type:timestamp with time zone" json:"scan_checked_at"`
	Revision         int16      `gorm:"not null;default:1" json:"revision"`
	SupersedesID     *uuid.UUID `gorm:"type:uuid;index" json:"supersedes_id"`
	SupersededByID   *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"superseded_by_id"`
	UploadedBy       uuid.UUID  `gorm:"type:uuid;not null" json:"uploaded_by"`
	Uploader         User       `gorm:"foreignKey:UploadedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PermohonanID     *uuid.UUID `gorm:"type:uuid;index" json:"permohonan_id"`
	Permohonan       Permohonan `gorm:"foreignKey:PermohonanID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

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
