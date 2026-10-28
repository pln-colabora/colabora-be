package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/notification/repository"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"
)

type reminderRepository struct {
	repository.EmailRepository
	request entities.Permohonan
	queued  []entities.EmailNotification
}

func (r *reminderRepository) FindOpenRequests(context.Context) ([]entities.Permohonan, error) {
	return []entities.Permohonan{r.request}, nil
}
func (r *reminderRepository) FindVerifiedRoleRecipients(context.Context, *gorm.DB, string, string) ([]entities.User, error) {
	return []entities.User{{ID: uuid.New(), Email: "synthetic@example.test", IsVerified: true}}, nil
}
func (r *reminderRepository) Enqueue(_ context.Context, _ *gorm.DB, rows []entities.EmailNotification) error {
	r.queued = append(r.queued, rows...)
	return nil
}
func TestWorkdayRemindersSkipWeekendAndHoliday(t *testing.T) {
	deadline := time.Date(2026, 6, 2, 0, 0, 0, 0, businessday.WIB)
	for _, test := range []struct {
		day   int
		month time.Month
		kind  string
	}{{29, 5, "sla_reminder"}, {30, 5, ""}, {31, 5, ""}, {1, 6, ""}, {2, 6, ""}, {3, 6, "sla_overdue"}} {
		repo := &reminderRepository{request: entities.Permohonan{ID: uuid.New(), JenisSambungan: "JTR", UlpUnit: "ULP Uji", WorkflowNodes: []entities.PermohonanActivity{{WorkflowNode: "permohonan", Status: "completed"}, {WorkflowNode: "survei", Status: "available", SlaDeadline: &deadline, SlaBasis: businessday.PaymentBasis}}}}
		service := &emailService{repository: repo, now: func() time.Time { return time.Date(2026, test.month, test.day, 9, 0, 0, 0, businessday.WIB) }}
		require.NoError(t, service.QueueDueSLA(context.Background()))
		if test.kind == "" {
			require.Empty(t, repo.queued)
		} else {
			require.Len(t, repo.queued, 1)
			require.True(t, strings.HasPrefix(repo.queued[0].DeduplicationKey, test.kind+"/"))
			if test.kind == "sla_reminder" {
				require.Contains(t, repo.queued[0].Body, "hari kerja berikutnya")
			}
		}
	}
}
