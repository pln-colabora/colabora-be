package service

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/notification/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

const (
	DefaultSLAInterval  = time.Minute
	DefaultSendInterval = 15 * time.Second
	claimBatchSize      = 25
	claimLease          = 2 * time.Minute
)

type Sender interface {
	SendMail(to, subject, body string) error
}

type EmailService interface {
	QueueInitialAvailable(context.Context, *gorm.DB, entities.Permohonan, workflow.Result) error
	QueueNewlyAvailable(context.Context, *gorm.DB, entities.Permohonan, workflow.Result, workflow.Result) error
	QueueVendorAssigned(context.Context, *gorm.DB, entities.Permohonan, entities.User) error
	QueueAccountRegistration(context.Context, *gorm.DB, entities.User) error
	QueueDueSLA(context.Context) error
	DispatchDue(context.Context) error
	Run(context.Context)
}

type emailService struct {
	repository repository.EmailRepository
	sender     Sender
	now        func() time.Time
}

func NewEmailService(repo repository.EmailRepository, sender Sender) EmailService {
	if sender == nil {
		sender = smtpSender{}
	}
	return &emailService{repository: repo, sender: sender, now: time.Now}
}

type smtpSender struct{}

func (smtpSender) SendMail(to, subject, body string) error {
	return utils.SendMail(to, subject, body)
}

func (s *emailService) QueueInitialAvailable(ctx context.Context, tx *gorm.DB, request entities.Permohonan, evaluated workflow.Result) error {
	for _, definition := range workflow.Definitions() {
		status := evaluated.Nodes[definition.Code]
		if isActionable(status) {
			if err := s.queueNode(ctx, tx, request, definition.Code, "activity"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *emailService) QueueNewlyAvailable(ctx context.Context, tx *gorm.DB, request entities.Permohonan, before, after workflow.Result) error {
	for _, definition := range workflow.Definitions() {
		if isActionable(before.Nodes[definition.Code]) || !isActionable(after.Nodes[definition.Code]) {
			continue
		}
		if err := s.queueNode(ctx, tx, request, definition.Code, "activity"); err != nil {
			return err
		}
	}
	return nil
}

func (s *emailService) QueueVendorAssigned(ctx context.Context, tx *gorm.DB, request entities.Permohonan, vendor entities.User) error {
	evaluated, err := workflow.Evaluate(request.WorkflowSnapshot())
	if err != nil {
		return err
	}
	for _, definition := range workflow.Definitions() {
		if !isActionable(evaluated.Nodes[definition.Code]) {
			continue
		}
		owner, ok := workflow.Owner(definition.Code, request.JenisSambungan)
		if !ok || owner != vendor.Role {
			continue
		}
		if err := s.queueNodeForUsers(ctx, tx, request, definition.Code, "activity", []entities.User{vendor}); err != nil {
			return err
		}
	}
	return nil
}

// QueueAccountRegistration alerts verified super-users after a registrant and
// their private account-verification document have been saved atomically.
func (s *emailService) QueueAccountRegistration(ctx context.Context, tx *gorm.DB, registrant entities.User) error {
	recipients, err := s.repository.FindVerifiedRoleRecipients(ctx, tx, rbac.RoleSuperUser, "")
	if err != nil {
		return err
	}

	subject := "COLABORA: Pendaftaran akun baru menunggu verifikasi"
	body := fmt.Sprintf(
		"<p>Ada pendaftaran akun baru yang menunggu verifikasi.</p><ul><li>Nama: %s</li><li>Email: %s</li><li>Terdaftar: %s</li></ul><p>Buka Manajemen Akun di COLABORA, filter <code>is_verified=false</code>, lalu periksa dokumen verifikasinya.</p>",
		html.EscapeString(registrant.Name),
		html.EscapeString(registrant.Email),
		registrant.CreatedAt.In(jakartaLocation()).Format("2006-01-02 15:04 WIB"),
	)
	now := s.now()
	rows := make([]entities.EmailNotification, 0, len(recipients))
	for _, recipient := range recipients {
		rows = append(rows, entities.EmailNotification{
			DeduplicationKey: fmt.Sprintf("account_registration/%s/%s", registrant.ID, recipient.ID),
			Recipient:        recipient.Email,
			Subject:          subject,
			Body:             body,
			Status:           repository.EmailPending,
			NextAttemptAt:    now,
		})
	}
	return s.repository.Enqueue(ctx, tx, rows)
}

func (s *emailService) QueueDueSLA(ctx context.Context) error {
	requests, err := s.repository.FindOpenRequests(ctx)
	if err != nil {
		return err
	}
	location := jakartaLocation()
	today := dateOnly(s.now().In(location), location)
	for _, request := range requests {
		evaluated, evaluateErr := workflow.Evaluate(request.WorkflowSnapshot())
		if evaluateErr != nil {
			return evaluateErr
		}
		for _, node := range request.WorkflowNodes {
			if node.SlaDeadline == nil || !isActionable(evaluated.Nodes[workflow.Code(node.WorkflowNode)]) {
				continue
			}
			deadline := dateOnly(*node.SlaDeadline, location)
			kind := ""
			if today.Equal(deadline.AddDate(0, 0, -1)) {
				kind = "sla_reminder"
			} else if !today.Before(deadline.AddDate(0, 0, 1)) {
				kind = "sla_overdue"
			}
			if kind == "" {
				continue
			}
			if err := s.queueNode(ctx, nil, request, workflow.Code(node.WorkflowNode), kind); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *emailService) queueNode(ctx context.Context, tx *gorm.DB, request entities.Permohonan, code workflow.Code, kind string) error {
	owner, ok := workflow.Owner(code, request.JenisSambungan)
	if !ok {
		return nil
	}
	var recipients []entities.User
	var err error
	if rbac.IsVendorRole(owner) {
		recipients, err = s.repository.FindAssignedVendorRecipients(ctx, tx, request.ID, owner)
	} else {
		recipients, err = s.repository.FindVerifiedRoleRecipients(ctx, tx, owner, request.UlpUnit)
	}
	if err != nil {
		return err
	}
	return s.queueNodeForUsers(ctx, tx, request, code, kind, recipients)
}

func (s *emailService) queueNodeForUsers(ctx context.Context, tx *gorm.DB, request entities.Permohonan, code workflow.Code, kind string, recipients []entities.User) error {
	node := findNode(request.WorkflowNodes, code)
	label := nodeLabel(code)
	var subject, body string
	switch kind {
	case "sla_reminder":
		subject = fmt.Sprintf("COLABORA: Reminder SLA - %s", label)
		body = fmt.Sprintf("<p>Pengingat: SLA untuk %s pada permohonan <strong>%s</strong> berakhir besok.</p>", html.EscapeString(label), html.EscapeString(request.NoPermohonan))
	case "sla_overdue":
		subject = fmt.Sprintf("COLABORA: SLA terlewati - %s", label)
		body = fmt.Sprintf("<p>SLA untuk %s pada permohonan <strong>%s</strong> telah terlewati.</p>", html.EscapeString(label), html.EscapeString(request.NoPermohonan))
	default:
		subject = fmt.Sprintf("COLABORA: Aktivitas baru - %s", label)
		body = fmt.Sprintf("<p>Aktivitas %s pada permohonan <strong>%s</strong> sekarang dapat dikerjakan.</p>", html.EscapeString(label), html.EscapeString(request.NoPermohonan))
	}
	if node.SlaDeadline != nil {
		body += fmt.Sprintf("<p>Deadline SLA: %s</p>", node.SlaDeadline.In(jakartaLocation()).Format("2006-01-02"))
	}
	body += "<p>Silakan buka COLABORA untuk melihat detail permohonan.</p>"

	now := s.now()
	rows := make([]entities.EmailNotification, 0, len(recipients))
	for _, user := range recipients {
		if !user.IsVerified || strings.TrimSpace(user.Email) == "" {
			continue
		}
		rows = append(rows, entities.EmailNotification{
			DeduplicationKey: fmt.Sprintf("%s/%s/%s/%s", kind, request.ID, code, user.ID),
			Recipient:        user.Email, Subject: subject, Body: body,
			Status: repository.EmailPending, NextAttemptAt: now,
		})
	}
	return s.repository.Enqueue(ctx, tx, rows)
}

func (s *emailService) DispatchDue(ctx context.Context) error {
	now := s.now()
	notifications, err := s.repository.ClaimDue(ctx, now, claimLease, claimBatchSize)
	if err != nil {
		return err
	}
	var firstErr error
	for _, notification := range notifications {
		if err := s.sender.SendMail(notification.Recipient, notification.Subject, notification.Body); err != nil {
			retryAt := now.Add(retryDelay(notification.Attempts))
			if markErr := s.repository.MarkFailed(ctx, notification.ID, notification.Attempts, err.Error(), retryAt); markErr != nil && firstErr == nil {
				firstErr = markErr
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := s.repository.MarkSent(ctx, notification.ID, now); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *emailService) Run(ctx context.Context) {
	if err := s.QueueDueSLA(ctx); err != nil {
		log.Printf("notification SLA scan failed: %v", err)
	}
	if err := s.DispatchDue(ctx); err != nil {
		log.Printf("notification dispatch failed: %v", err)
	}
	slaTicker := time.NewTicker(DefaultSLAInterval)
	sendTicker := time.NewTicker(DefaultSendInterval)
	defer slaTicker.Stop()
	defer sendTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-slaTicker.C:
			if err := s.QueueDueSLA(ctx); err != nil {
				log.Printf("notification SLA scan failed: %v", err)
			}
		case <-sendTicker.C:
			if err := s.DispatchDue(ctx); err != nil {
				log.Printf("notification dispatch failed: %v", err)
			}
		}
	}
}

func isActionable(status workflow.Status) bool {
	return status == workflow.Available || status == workflow.InProgress
}

func dateOnly(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func jakartaLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return location
}

func findNode(nodes []entities.PermohonanActivity, code workflow.Code) entities.PermohonanActivity {
	for _, node := range nodes {
		if node.WorkflowNode == string(code) {
			return node
		}
	}
	return entities.PermohonanActivity{}
}

func nodeLabel(code workflow.Code) string {
	definition, ok := workflow.Lookup(code)
	if !ok || definition.ActivityNumber == nil {
		return string(code)
	}
	return fmt.Sprintf("Aktivitas #%d", *definition.ActivityNumber)
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	return time.Duration(1<<uint(attempt-1)) * time.Minute
}
