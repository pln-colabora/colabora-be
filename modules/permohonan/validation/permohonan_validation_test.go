package validation

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
)

func TestValidatePermohonanPowerFields(t *testing.T) {
	v := NewPermohonanValidation()
	newPower := int64(2200)
	oldPower := int64(1300)

	base := dto.PermohonanCreateRequest{
		JenisPermohonan: rbac.JenisPermohonanPasangBaru,
		JenisSambungan:  rbac.JenisSambunganJTR,
		Tarif:           stringPointer(rbac.TarifRumahTangga),
		DayaBaru:        &newPower,
		PelangganNama:   "Pelanggan Uji",
		PelangganAlamat: "Alamat Uji",
		PelangganNoHp:   "0800000000",
	}
	require.NoError(t, v.ValidatePermohonanCreateRequest(base))
	legacyShape := base
	legacyShape.Tarif = nil
	legacyShape.DayaBaru = nil
	require.NoError(t, v.ValidatePermohonanCreateRequest(legacyShape))

	pd := base
	pd.JenisPermohonan = rbac.JenisPermohonanPerubahanDaya
	pd.DayaLama = &oldPower
	require.NoError(t, v.ValidatePermohonanCreateRequest(pd))

	invalidPB := base
	invalidPB.DayaLama = &oldPower
	require.Error(t, v.ValidatePermohonanCreateRequest(invalidPB))

	invalidPD := pd
	invalidPD.DayaBaru = &oldPower
	require.Error(t, v.ValidatePermohonanCreateRequest(invalidPD))

	invalidTariff := base
	invalidTariff.Tarif = stringPointer("lainnya")
	require.Error(t, v.ValidatePermohonanCreateRequest(invalidTariff))
}

func stringPointer(value string) *string { return &value }

func TestValidateSequenceOneRequests(t *testing.T) {
	v := NewPermohonanValidation()
	documentID := uuid.NewString()
	poleRequired := false

	require.NoError(t, v.ValidateSurveySubmitRequest(dto.SurveySubmitRequest{
		SurveyedAt: "2026-09-08", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateSurveySubmitRequest(dto.SurveySubmitRequest{
		SurveyedAt: "08-09-2026", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateSurveySubmitRequest(dto.SurveySubmitRequest{
		SurveyedAt: "2026-09-08", DocumentIDs: []string{documentID, documentID},
	}))

	require.NoError(t, v.ValidateRABSubmitRequest(dto.RABSubmitRequest{
		KebutuhanTiang: &poleRequired, DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateRABSubmitRequest(dto.RABSubmitRequest{
		DocumentIDs: []string{documentID},
	}))

	require.NoError(t, v.ValidateExpansionSubmitRequest(dto.ExpansionSubmitRequest{
		NpsDelegationStatus: "delegated", DocumentIDs: []string{documentID},
	}))
	require.NoError(t, v.ValidateExpansionSubmitRequest(dto.ExpansionSubmitRequest{
		NpsDelegationStatus: "returned", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateExpansionSubmitRequest(dto.ExpansionSubmitRequest{
		NpsDelegationStatus: "approved", DocumentIDs: []string{documentID},
	}))
}

func TestValidateSequenceTwoRequests(t *testing.T) {
	v := NewPermohonanValidation()
	documentID := uuid.NewString()
	pdkbRequired := false
	latitude, longitude := -6.2, 106.816666

	require.NoError(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		Notes: "synthetic evidence", DocumentIDs: []string{documentID},
		LocationCoordinates: &dto.LocationCoordinates{Latitude: &latitude, Longitude: &longitude},
	}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID}, LocationCoordinates: &dto.LocationCoordinates{Latitude: &latitude},
	}))
	invalidLatitude := 91.0
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID}, LocationCoordinates: &dto.LocationCoordinates{Latitude: &invalidLatitude, Longitude: &longitude},
	}))
	invalidLongitude := 181.0
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID}, LocationCoordinates: &dto.LocationCoordinates{Latitude: &latitude, Longitude: &invalidLongitude},
	}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID, documentID},
	}))

	require.NoError(t, v.ValidateWOConstructionSubmitRequest(dto.WOConstructionSubmitRequest{
		PerluPdkb: &pdkbRequired, DocumentIDs: []string{documentID},
		LocationCoordinates: &dto.LocationCoordinates{Latitude: &latitude, Longitude: &longitude},
	}))
	require.Error(t, v.ValidateWOConstructionSubmitRequest(dto.WOConstructionSubmitRequest{
		DocumentIDs: []string{documentID},
	}))

	require.NoError(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{"not-a-uuid"},
	}))
}

func TestValidateSequenceThreeExecutionSelector(t *testing.T) {
	v := NewPermohonanValidation()
	documentIDs := []string{uuid.NewString()}

	for _, node := range []string{"pemasangan_tiang", "pelaksanaan_konstruksi"} {
		require.NoError(t, v.ValidateConstructionExecutionSubmitRequest(dto.ConstructionExecutionSubmitRequest{
			WorkflowNode: node, DocumentIDs: documentIDs,
		}))
	}
	require.Error(t, v.ValidateConstructionExecutionSubmitRequest(dto.ConstructionExecutionSubmitRequest{
		WorkflowNode: "wo_tiang", DocumentIDs: documentIDs,
	}))
	require.Error(t, v.ValidateConstructionExecutionSubmitRequest(dto.ConstructionExecutionSubmitRequest{
		WorkflowNode: "pemasangan_tiang",
	}))
}

func TestValidateSequenceFourRequests(t *testing.T) {
	v := NewPermohonanValidation()
	documentID := uuid.NewString()

	require.NoError(t, v.ValidateEnergizeSubmitRequest(dto.EnergizeSubmitRequest{
		OperationResult: "operasi jaringan selesai", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateEnergizeSubmitRequest(dto.EnergizeSubmitRequest{
		OperationResult: "   ", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateEnergizeSubmitRequest(dto.EnergizeSubmitRequest{
		OperationResult: "operasi jaringan selesai", DocumentIDs: []string{documentID, documentID},
	}))
}
