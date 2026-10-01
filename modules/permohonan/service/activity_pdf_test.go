package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateActivityPDFRendersDynamicPayload(t *testing.T) {
	number := int16(2)
	content, err := generateActivityPDF(activityPDFData{
		NoPermohonan:   "PB-2026-00001",
		JenisSambungan: "JTR",
		WorkflowNode:   "survei",
		ActivityNumber: &number,
		StageNumber:    2,
		Status:         "completed",
		Payload:        json.RawMessage(`{"notes":"survei – kondisi aman","alat_ukur":[{"nama":"meter-a","hasil":12.5}],"verified":true}`),
	})

	require.NoError(t, err)
	require.Greater(t, len(content), 100)
	require.Equal(t, "%PDF", string(content[:4]))
}

func TestGenerateActivityPDFHandlesEmptyPayload(t *testing.T) {
	content, err := generateActivityPDF(activityPDFData{
		WorkflowNode: "survei",
		Payload:      json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(content[:4]))
}

func TestGenerateActivityPDFRejectsMalformedPayload(t *testing.T) {
	_, err := generateActivityPDF(activityPDFData{Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}
