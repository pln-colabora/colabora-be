package service

import (
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"time"
)

func nodeRemaining(node entities.PermohonanActivity, now time.Time) (*int, string) {
	if node.SlaDeadline == nil || node.Status == "skipped" {
		return nil, ""
	}
	if node.CompletedAt != nil {
		now = *node.CompletedAt
	}
	if node.SlaBasis == businessday.PaymentBasis {
		days, err := businessday.Remaining(*node.SlaDeadline, now)
		if err != nil {
			return nil, err.Error()
		}
		return &days, ""
	}
	deadline := time.Date(node.SlaDeadline.Year(), node.SlaDeadline.Month(), node.SlaDeadline.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	days := int(deadline.Sub(today).Hours() / 24)
	return &days, ""
}
