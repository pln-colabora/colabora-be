package dto

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrRevisionConflict = errors.New("data sudah berubah; muat ulang sebelum menyimpan")
var ErrScheduleConflict = errors.New("tanggal PDKB sudah terisi; pilih tanggal lain")
var ErrScheduleLocked = errors.New("booking WO tidak dapat diubah melalui kalender")

// Priority values are case-sensitive and stored verbatim on the node payload
// and the permohonan projection.
var PriorityValues = []string{"High", "Medium", "Low"}

type PrioritySubmitRequest struct {
	Priority    string   `json:"priority"`
	Notes       string   `json:"notes" binding:"omitempty,max=2000"`
	DocumentIDs []string `json:"document_ids,omitempty" binding:"omitempty,dive,uuid"`
}
type SurveyRevisionRequest struct {
	SurveyedAt       string   `json:"surveyed_at" binding:"required"`
	Notes            string   `json:"notes" binding:"omitempty,max=2000"`
	DocumentIDs      []string `json:"document_ids" binding:"required,min=1,dive,uuid"`
	ExpectedRevision int      `json:"expected_revision" binding:"required,min=1"`
}
type DocumentRevisionRequest struct {
	DocumentIDs      []string `json:"document_ids" binding:"required,min=1,dive,uuid"`
	ExpectedRevision int      `json:"expected_revision" binding:"required,min=1"`
}
type ActivityRevisionResponse struct {
	Revision    int             `json:"revision"`
	Payload     json.RawMessage `json:"payload"`
	DocumentIDs []string        `json:"document_ids"`
	RecordedBy  uuid.UUID       `json:"recorded_by"`
	CreatedAt   time.Time       `json:"created_at"`
}
type PDKBScheduleRequest struct {
	Date  string `json:"date" binding:"required"`
	Title string `json:"title" binding:"required,min=1,max=200"`
}
type PDKBScheduleResponse struct {
	ID           string     `json:"id"`
	Date         string     `json:"date"`
	Title        string     `json:"title"`
	PermohonanID *uuid.UUID `json:"permohonan_id"`
	Locked       bool       `json:"locked"`
}
