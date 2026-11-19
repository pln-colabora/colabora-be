package service

import (
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
)

// hplDays counts working days after the payment date up to today, so paying on
// Mon 10 Nov and viewing on Tue 18 Nov gives 6. It stops once the request is
// terminal: at the "selesai" node's completion when finished, or at the NPS
// delegation's completion when returned. Nil means no payment date yet or a date
// outside the supported calendar.
func hplDays(payment *string, status workflow.AggregateStatus, nodes []entities.PermohonanActivity, updatedAt string, now time.Time) *int {
	if payment == nil {
		return nil
	}
	paid, err := time.ParseInLocation("2006-01-02", *payment, businessday.WIB)
	if err != nil {
		return nil
	}
	end := now
	switch status {
	case workflow.Finished:
		end = stoppedAt(nodes, workflow.Selesai, updatedAt, now)
	case workflow.Returned:
		end = stoppedAt(nodes, workflow.NPS, updatedAt, now)
	}
	if businessday.Date(end).Before(paid) {
		zero := 0
		return &zero
	}
	// Remaining counts (earlier date, later date], which is exactly the HPL span.
	days, err := businessday.Remaining(end, paid)
	if err != nil {
		return nil
	}
	return &days
}

func stoppedAt(nodes []entities.PermohonanActivity, code workflow.Code, updatedAt string, fallback time.Time) time.Time {
	for _, node := range nodes {
		if node.WorkflowNode == string(code) && node.CompletedAt != nil {
			return *node.CompletedAt
		}
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		return parsed
	}
	return fallback
}
