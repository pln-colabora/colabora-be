package service

import (
	"testing"
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestHplDays(t *testing.T) {
	at := func(date string) time.Time {
		v, err := time.ParseInLocation("2006-01-02", date, businessday.WIB)
		require.NoError(t, err)
		return v.Add(10 * time.Hour)
	}
	str := func(s string) *string { return &s }
	completedNode := func(code workflow.Code, date string) []entities.PermohonanActivity {
		completed := at(date)
		return []entities.PermohonanActivity{{WorkflowNode: string(code), CompletedAt: &completed}}
	}
	finished := func(date string) []entities.PermohonanActivity { return completedNode(workflow.Selesai, date) }
	returned := func(date string) []entities.PermohonanActivity { return completedNode(workflow.NPS, date) }
	tests := []struct {
		name    string
		payment *string
		status  workflow.AggregateStatus
		nodes   []entities.PermohonanActivity
		now     string
		want    *int
	}{
		{"example: paid Mon 10 Nov, today Tue 18 Nov", str("2026-11-10"), workflow.Active, nil, "2026-11-18", new(6)},
		{"paid on Friday, today Monday skips the weekend", str("2026-11-13"), workflow.Active, nil, "2026-11-16", new(1)},
		{"paid on Saturday starts counting Monday", str("2026-11-14"), workflow.Active, nil, "2026-11-16", new(1)},
		{"same day is zero", str("2026-11-10"), workflow.Active, nil, "2026-11-10", new(0)},
		{"national holiday is skipped", str("2026-08-14"), workflow.Active, nil, "2026-08-18", new(1)},
		{"stops at completion of selesai", str("2026-11-10"), workflow.Finished, finished("2026-11-12"), "2026-11-18", new(2)},
		{"stops at NPS return", str("2026-11-10"), workflow.Returned, returned("2026-11-13"), "2026-11-18", new(3)},
		{"no payment date", nil, workflow.Active, nil, "2026-11-18", nil},
		{"unsupported calendar year", str("2025-12-01"), workflow.Active, nil, "2026-11-18", nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, hplDays(test.payment, test.status, test.nodes, "", at(test.now)))
		})
	}
}
