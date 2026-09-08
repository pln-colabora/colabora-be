package validation

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/stretchr/testify/require"
)

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

	require.NoError(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		Notes: "synthetic evidence", DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{}))
	require.Error(t, v.ValidateEvidenceSubmitRequest(dto.EvidenceSubmitRequest{
		DocumentIDs: []string{documentID, documentID},
	}))

	require.NoError(t, v.ValidateWOConstructionSubmitRequest(dto.WOConstructionSubmitRequest{
		PerluPdkb: &pdkbRequired, DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateWOConstructionSubmitRequest(dto.WOConstructionSubmitRequest{
		DocumentIDs: []string{documentID},
	}))

	require.NoError(t, v.ValidateReservationTeraSubmitRequest(dto.ReservationTeraSubmitRequest{
		DocumentIDs: []string{documentID},
	}))
	require.Error(t, v.ValidateReservationTeraSubmitRequest(dto.ReservationTeraSubmitRequest{
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
