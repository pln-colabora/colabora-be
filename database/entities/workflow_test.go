package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestInitializeWorkflowPersistsEveryCanonicalNode(t *testing.T) {
	requestDate := time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC)
	permohonan := Permohonan{
		ID:             uuid.New(),
		CreatedBy:      uuid.New(),
		JenisSambungan: "JTR",
		RequestDate:    requestDate,
	}
	rules := make([]SLARule, 0)
	for _, definition := range workflow.Definitions() {
		if definition.SLAActivityNumber != nil {
			rules = append(rules, SLARule{
				ActivityNumber: *definition.SLAActivityNumber,
				JenisSambungan: permohonan.JenisSambungan,
				OffsetDays:     2,
			})
		}
	}

	nodes, result, err := InitializeWorkflow(permohonan, rules, requestDate.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, nodes, len(workflow.Definitions()))
	require.Equal(t, workflow.Active, result.Status)

	byCode := make(map[string]PermohonanActivity, len(nodes))
	for _, node := range nodes {
		byCode[node.WorkflowNode] = node
	}
	require.Equal(t, "completed", byCode[string(workflow.Permohonan)].Status)
	require.NotNil(t, byCode[string(workflow.Permohonan)].CompletedBy)
	require.Equal(t, "available", byCode[string(workflow.Survei)].Status)
	require.Nil(t, byCode[string(workflow.KebutuhanTiang)].ActivityNumber)
	require.Nil(t, byCode[string(workflow.KebutuhanTiang)].SlaDeadline)
	require.NotNil(t, byCode[string(workflow.Survei)].SlaDeadline)
	require.NotContains(t, byCode, string(workflow.PKVendor))
}

func TestWorkflowSnapshotIgnoresRetiredPKVendorHistory(t *testing.T) {
	p := Permohonan{
		JenisSambungan: "JTR",
		WorkflowNodes:  []PermohonanActivity{{WorkflowNode: string(workflow.PKVendor), Status: string(workflow.Completed)}},
	}

	result, err := workflow.Evaluate(p.WorkflowSnapshot())
	require.NoError(t, err)
	require.NotContains(t, result.Nodes, workflow.PKVendor)
}

func TestInitializeWorkflowRejectsMissingSLARule(t *testing.T) {
	_, _, err := InitializeWorkflow(Permohonan{
		ID:             uuid.New(),
		CreatedBy:      uuid.New(),
		JenisSambungan: "JTR",
		RequestDate:    time.Now(),
	}, nil, time.Now())
	require.Error(t, err)
}
