package dto

import (
	"encoding/json"
	"errors"
)

const (
	MESSAGE_FAILED_GET_DATA_FROM_BODY  = "failed get data from body"
	MESSAGE_FAILED_CREATE_PERMOHONAN   = "failed create permohonan"
	MESSAGE_FAILED_GET_PERMOHONAN      = "failed get permohonan"
	MESSAGE_FAILED_GET_LIST_PERMOHONAN = "failed get list permohonan"
	MESSAGE_FAILED_GET_ACTIVITIES      = "failed get workflow activities"
	MESSAGE_FAILED_GET_LOGS            = "failed get activity logs"
	MESSAGE_FAILED_SUBMIT_ACTIVITY     = "failed submit workflow activity"
	MESSAGE_FAILED_DENIED_ACCESS       = "denied access"

	MESSAGE_SUCCESS_CREATE_PERMOHONAN   = "success create permohonan"
	MESSAGE_SUCCESS_GET_PERMOHONAN      = "success get permohonan"
	MESSAGE_SUCCESS_GET_LIST_PERMOHONAN = "success get list permohonan"
	MESSAGE_SUCCESS_GET_ACTIVITIES      = "success get workflow activities"
	MESSAGE_SUCCESS_GET_LOGS            = "success get activity logs"
	MESSAGE_SUCCESS_SUBMIT_ACTIVITY     = "success submit workflow activity"
)

var (
	ErrCreatePermohonan   = errors.New("failed to create permohonan")
	ErrGetPermohonanById  = errors.New("failed to get permohonan by id")
	ErrPermohonanNotFound = errors.New("permohonan not found")
	ErrCreateForbidden    = errors.New("caller cannot create this connection type")
	ErrInvalidULPUnit     = errors.New("invalid ulp_unit for this connection type")
	ErrEvidenceRequired   = errors.New("at least one evidence document is required")
	ErrInvalidEvidence    = errors.New("document_ids must contain unique UUIDs")
	ErrInvalidActivity    = errors.New("invalid workflow activity payload")
	ErrSubmitActivity     = errors.New("failed to submit workflow activity")
)

type (
	PermohonanCreateRequest struct {
		JenisPermohonan string  `json:"jenis_permohonan" binding:"required"`
		JenisSambungan  string  `json:"jenis_sambungan" binding:"required"`
		UlpUnit         *string `json:"ulp_unit"`
		PelangganNama   string  `json:"pelanggan_nama" binding:"required,min=2,max=150"`
		PelangganAlamat string  `json:"pelanggan_alamat" binding:"required,min=2,max=255"`
		PelangganNoHp   string  `json:"pelanggan_no_hp" binding:"required,min=8,max=20"`
	}

	SurveySubmitRequest struct {
		SurveyedAt  string   `json:"surveyed_at" binding:"required" validate:"required"`
		Notes       string   `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs []string `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
	}

	RABSubmitRequest struct {
		KebutuhanTiang *bool    `json:"kebutuhan_tiang" binding:"required" validate:"required"`
		Notes          string   `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs    []string `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
	}

	ExpansionSubmitRequest struct {
		NpsDelegationStatus string   `json:"nps_delegation_status" binding:"required" validate:"required"`
		Notes               string   `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs         []string `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
	}

	PermohonanResponse struct {
		ID                  string                 `json:"id"`
		NoPermohonan        string                 `json:"no_permohonan"`
		JenisPermohonan     string                 `json:"jenis_permohonan"`
		JenisSambungan      string                 `json:"jenis_sambungan"`
		UlpUnit             string                 `json:"ulp_unit"`
		PelangganNama       string                 `json:"pelanggan_nama"`
		PelangganAlamat     string                 `json:"pelanggan_alamat"`
		PelangganNoHp       string                 `json:"pelanggan_no_hp"`
		RequestDate         string                 `json:"request_date"`
		CurrentStage        int16                  `json:"current_stage"`
		Status              string                 `json:"status"`
		KebutuhanTiang      *bool                  `json:"kebutuhan_tiang"`
		NpsDelegationStatus *string                `json:"nps_delegation_status"`
		PerluPdkb           *bool                  `json:"perlu_pdkb"`
		CreatedBy           string                 `json:"created_by"`
		WorkflowNodes       []WorkflowNodeResponse `json:"workflow_nodes"`
		AvailableActions    []AvailableAction      `json:"available_actions"`
	}

	WorkflowNodeResponse struct {
		WorkflowNode   string          `json:"workflow_node"`
		ActivityNumber *int16          `json:"activity_number"`
		StageNumber    int16           `json:"stage_number"`
		Status         string          `json:"status"`
		SlaDeadline    *string         `json:"sla_deadline"`
		SlaStatus      string          `json:"sla_status"`
		Payload        json.RawMessage `json:"payload"`
		CompletedBy    *string         `json:"completed_by"`
		CompletedAt    *string         `json:"completed_at"`
	}

	AvailableAction struct {
		WorkflowNode   string `json:"workflow_node"`
		ActivityNumber *int16 `json:"activity_number"`
		StageNumber    int16  `json:"stage_number"`
		Method         string `json:"method"`
		Path           string `json:"path"`
	}

	ActivityLogResponse struct {
		ID             string  `json:"id"`
		WorkflowNode   *string `json:"workflow_node"`
		ActivityNumber *int16  `json:"activity_number"`
		Actor          string  `json:"actor"`
		Action         string  `json:"action"`
		Detail         *string `json:"detail"`
		CreatedAt      string  `json:"created_at"`
	}
)
