package dto

import (
	"encoding/json"
	"errors"
	"mime/multipart"

	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
)

const (
	MESSAGE_FAILED_GET_DATA_FROM_BODY  = "failed get data from body"
	MESSAGE_FAILED_CREATE_PERMOHONAN   = "failed create permohonan"
	MESSAGE_FAILED_GET_PERMOHONAN      = "failed get permohonan"
	MESSAGE_FAILED_GET_LIST_PERMOHONAN = "failed get list permohonan"
	MESSAGE_FAILED_GET_ACTIVITIES      = "failed get workflow activities"
	MESSAGE_FAILED_GET_ACTIVITY        = "failed get workflow activity"
	MESSAGE_FAILED_GET_LOGS            = "failed get activity logs"
	MESSAGE_FAILED_SUBMIT_ACTIVITY     = "failed submit workflow activity"
	MESSAGE_FAILED_DENIED_ACCESS       = "denied access"

	MESSAGE_SUCCESS_CREATE_PERMOHONAN   = "success create permohonan"
	MESSAGE_SUCCESS_GET_PERMOHONAN      = "success get permohonan"
	MESSAGE_SUCCESS_GET_LIST_PERMOHONAN = "success get list permohonan"
	MESSAGE_SUCCESS_GET_ACTIVITIES      = "success get workflow activities"
	MESSAGE_SUCCESS_GET_ACTIVITY        = "success get workflow activity"
	MESSAGE_SUCCESS_GET_LOGS            = "success get activity logs"
	MESSAGE_SUCCESS_SUBMIT_ACTIVITY     = "success submit workflow activity"
)

var (
	ErrCreatePermohonan     = errors.New("failed to create permohonan")
	ErrGetPermohonanById    = errors.New("failed to get permohonan by id")
	ErrPermohonanNotFound   = errors.New("permohonan not found")
	ErrWorkflowNodeNotFound = errors.New("workflow node not found")
	ErrCreateForbidden      = errors.New("caller cannot create this connection type")
	ErrInvalidULPUnit       = errors.New("invalid ulp_unit for this connection type")
	ErrInvalidTariffPower   = errors.New("daya is not available for the selected tariff and connection type")
	ErrEvidenceRequired     = errors.New("at least one evidence document is required")
	ErrInvalidEvidence      = errors.New("document_ids must contain unique UUIDs")
	ErrInvalidActivity      = errors.New("invalid workflow activity payload")
	ErrSubmitActivity       = errors.New("failed to submit workflow activity")
)

type (
	LocationCoordinates struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}

	VendorAssignmentRequest struct {
		VendorID   string `json:"vendor_id" binding:"required,uuid" validate:"required,uuid"`
		VendorRole string `json:"vendor_role" binding:"required,oneof=vendor-tiang vendor-konstruksi vendor-sr-app" validate:"required,oneof=vendor-tiang vendor-konstruksi vendor-sr-app"`
	}
	PermohonanCreateRequest struct {
		JenisPermohonan string                  `json:"jenis_permohonan" form:"jenis_permohonan" binding:"required"`
		JenisSambungan  string                  `json:"jenis_sambungan" form:"jenis_sambungan" binding:"required"`
		Tarif           *string                 `json:"tarif" form:"tarif"`
		DayaLama        *int64                  `json:"daya_lama" form:"daya_lama"`
		DayaBaru        *int64                  `json:"daya_baru" form:"daya_baru"`
		UlpUnit         *string                 `json:"ulp_unit" form:"ulp_unit"`
		PelangganNama   string                  `json:"pelanggan_nama" form:"pelanggan_nama" binding:"required,min=2,max=150"`
		PelangganAlamat string                  `json:"pelanggan_alamat" form:"pelanggan_alamat" binding:"required,min=2,max=255"`
		PelangganNoHp   string                  `json:"pelanggan_no_hp" form:"pelanggan_no_hp" binding:"required,min=8,max=20"`
		EvidenceFiles   []*multipart.FileHeader `json:"-" form:"evidence_files"`
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

	EvidenceSubmitRequest struct {
		Notes               string               `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs         []string             `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
		LocationCoordinates *LocationCoordinates `json:"location_coordinates,omitempty"`
	}

	WOConstructionSubmitRequest struct {
		EstimasiTanggalSelesai string               `json:"estimasi_tanggal_selesai" binding:"required" validate:"required"`
		PerluPdkb              *bool                `json:"perlu_pdkb" binding:"required" validate:"required"`
		Notes                  string               `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs            []string             `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
		LocationCoordinates    *LocationCoordinates `json:"location_coordinates,omitempty"`
	}

	ConstructionExecutionSubmitRequest struct {
		WorkflowNode        string               `json:"workflow_node" binding:"required" validate:"required"`
		Notes               string               `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs         []string             `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
		LocationCoordinates *LocationCoordinates `json:"location_coordinates,omitempty"`
	}

	EnergizeSubmitRequest struct {
		OperationResult string   `json:"operation_result" binding:"required,max=500" validate:"required,max=500"`
		Notes           string   `json:"notes" binding:"omitempty,max=2000" validate:"omitempty,max=2000"`
		DocumentIDs     []string `json:"document_ids" binding:"required,min=1,dive,uuid" validate:"required,min=1,dive,uuid"`
	}

	PermohonanResponse struct {
		ID                  string                 `json:"id"`
		NoPermohonan        string                 `json:"no_permohonan"`
		JenisPermohonan     string                 `json:"jenis_permohonan"`
		JenisSambungan      string                 `json:"jenis_sambungan"`
		Tarif               *string                `json:"tarif"`
		DayaLama            *int64                 `json:"daya_lama"`
		DayaBaru            *int64                 `json:"daya_baru"`
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

	TariffPowerOptionResponse struct {
		Tarif          string `json:"tarif"`
		GolonganTarif  string `json:"golongan_tarif"`
		JenisSambungan string `json:"jenis_sambungan"`
		DayaMin        int64  `json:"daya_min"`
		DayaMax        *int64 `json:"daya_max"`
		Label          string `json:"label"`
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

	WorkflowNodeDetailResponse struct {
		WorkflowNodeResponse
		Documents []documentDTO.DocumentResponse `json:"documents"`
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
