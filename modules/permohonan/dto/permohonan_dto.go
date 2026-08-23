package dto

import "errors"

const (
	MESSAGE_FAILED_GET_DATA_FROM_BODY  = "failed get data from body"
	MESSAGE_FAILED_CREATE_PERMOHONAN   = "failed create permohonan"
	MESSAGE_FAILED_GET_PERMOHONAN      = "failed get permohonan"
	MESSAGE_FAILED_GET_LIST_PERMOHONAN = "failed get list permohonan"
	MESSAGE_FAILED_DENIED_ACCESS       = "denied access"

	MESSAGE_SUCCESS_CREATE_PERMOHONAN   = "success create permohonan"
	MESSAGE_SUCCESS_GET_PERMOHONAN      = "success get permohonan"
	MESSAGE_SUCCESS_GET_LIST_PERMOHONAN = "success get list permohonan"
)

var (
	ErrCreatePermohonan                = errors.New("failed to create permohonan")
	ErrGetPermohonanById               = errors.New("failed to get permohonan by id")
	ErrPermohonanNotFound              = errors.New("permohonan not found")
	ErrOnlyPelayananPelangganCanCreate = errors.New("only pelayanan-pelanggan can create a permohonan")
)

type (
	PermohonanCreateRequest struct {
		JenisPermohonan string `json:"jenis_permohonan" binding:"required"`
		JenisSambungan  string `json:"jenis_sambungan" binding:"required"`
		PelangganNama   string `json:"pelanggan_nama" binding:"required,min=2,max=150"`
		PelangganAlamat string `json:"pelanggan_alamat" binding:"required,min=2,max=255"`
		PelangganNoHp   string `json:"pelanggan_no_hp" binding:"required,min=8,max=20"`
	}

	PermohonanResponse struct {
		ID              string  `json:"id"`
		NoPermohonan    string  `json:"no_permohonan"`
		JenisPermohonan string  `json:"jenis_permohonan"`
		JenisSambungan  string  `json:"jenis_sambungan"`
		UlpUnit         string  `json:"ulp_unit"`
		PelangganNama   string  `json:"pelanggan_nama"`
		PelangganAlamat string  `json:"pelanggan_alamat"`
		PelangganNoHp   string  `json:"pelanggan_no_hp"`
		RequestDate     string  `json:"request_date"`
		CurrentStage    int16   `json:"current_stage"`
		Status          string  `json:"status"`
		KebutuhanTiang  *bool   `json:"kebutuhan_tiang"`
		NpsKeputusan    *string `json:"nps_keputusan"`
		PerluPdkb       *bool   `json:"perlu_pdkb"`
		OwnerFnOverride string  `json:"owner_fn_override"`
		CreatedBy       string  `json:"created_by"`
		CanAct          bool    `json:"can_act"`
	}
)
