package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permohonan struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NoPermohonan        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"no_permohonan"`
	JenisPermohonan     string    `gorm:"type:varchar(30);not null" json:"jenis_permohonan"`
	JenisSambungan      string    `gorm:"type:varchar(30);not null" json:"jenis_sambungan"`
	UlpUnit             string    `gorm:"type:varchar(50);not null" json:"ulp_unit"`
	PelangganNama       string    `gorm:"type:varchar(150);not null" json:"pelanggan_nama"`
	PelangganAlamat     string    `gorm:"type:varchar(255);not null" json:"pelanggan_alamat"`
	PelangganNoHp       string    `gorm:"type:varchar(20);not null" json:"pelanggan_no_hp"`
	RequestDate         time.Time `gorm:"type:date;not null" json:"request_date"`
	CurrentStage        int16     `gorm:"type:smallint;not null;default:1" json:"current_stage"`
	Status              string    `gorm:"type:varchar(20);not null;default:'in_progress'" json:"status"`
	KebutuhanTiang      *bool     `json:"kebutuhan_tiang"`
	NpsDelegationStatus *string   `gorm:"type:varchar(20)" json:"nps_delegation_status"`
	PerluPdkb           *bool     `json:"perlu_pdkb"`
	CreatedBy           uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	Creator             User      `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	WorkflowNodes []PermohonanActivity `gorm:"foreignKey:PermohonanID" json:"-"`
	Timestamp
}

func (Permohonan) TableName() string {
	return "permohonan"
}

func (p *Permohonan) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
