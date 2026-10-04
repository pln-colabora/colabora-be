package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EmailPending = "pending"
	EmailSending = "sending"
	EmailSent    = "sent"
)

type EmailRepository interface {
	Enqueue(context.Context, *gorm.DB, []entities.EmailNotification) error
	FindVerifiedRoleRecipients(context.Context, *gorm.DB, string, string) ([]entities.User, error)
	FindAssignedVendorRecipients(context.Context, *gorm.DB, uuid.UUID, string) ([]entities.User, error)
	FindOpenRequests(context.Context) ([]entities.Permohonan, error)
	ClaimDue(context.Context, time.Time, time.Duration, int) ([]entities.EmailNotification, error)
	MarkSent(context.Context, uuid.UUID, time.Time) error
	MarkFailed(context.Context, uuid.UUID, int, string, time.Time) error
}

type emailRepository struct{ db *gorm.DB }

func NewEmailRepository(db *gorm.DB) EmailRepository { return &emailRepository{db: db} }

func (r *emailRepository) Enqueue(ctx context.Context, tx *gorm.DB, notifications []entities.EmailNotification) error {
	if len(notifications) == 0 {
		return nil
	}
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "deduplication_key"}}, DoNothing: true}).Create(&notifications).Error
}

func (r *emailRepository) FindVerifiedRoleRecipients(ctx context.Context, tx *gorm.DB, role, ulp string) ([]entities.User, error) {
	if tx == nil {
		tx = r.db
	}
	query := tx.WithContext(ctx).Where("role = ? AND is_verified = ? AND email <> ''", role, true)
	if role == rbac.RoleTeknik || role == rbac.RolePelayananPelanggan {
		query = query.Where("unit = ?", ulp)
	}
	var users []entities.User
	err := query.Order("id").Find(&users).Error
	return users, err
}

func (r *emailRepository) FindAssignedVendorRecipients(ctx context.Context, tx *gorm.DB, permohonanID uuid.UUID, role string) ([]entities.User, error) {
	if tx == nil {
		tx = r.db
	}
	var users []entities.User
	err := tx.WithContext(ctx).Model(&entities.User{}).
		Joins("JOIN vendor_assignments ON vendor_assignments.vendor_id = users.id").
		Where("vendor_assignments.permohonan_id = ? AND vendor_assignments.vendor_role = ?", permohonanID, role).
		Where("users.is_verified = ? AND users.email <> ''", true).
		Order("users.id").Find(&users).Error
	return users, err
}

func (r *emailRepository) FindOpenRequests(ctx context.Context) ([]entities.Permohonan, error) {
	var requests []entities.Permohonan
	err := r.db.WithContext(ctx).
		Preload("WorkflowNodes").
		Where("status = ?", "in_progress").
		Order("id").Find(&requests).Error
	return requests, err
}

func (r *emailRepository) ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]entities.EmailNotification, error) {
	var claimed []entities.EmailNotification
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND locked_until <= ?)", EmailPending, now, EmailSending, now).
			Order("created_at, id").Limit(limit).Find(&claimed).Error; err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(claimed))
		for i := range claimed {
			ids = append(ids, claimed[i].ID)
			claimed[i].Status = EmailSending
			claimed[i].Attempts++
			lockedUntil := now.Add(lease)
			claimed[i].LockedUntil = &lockedUntil
		}
		if len(ids) == 0 {
			return nil
		}
		return tx.Model(&entities.EmailNotification{}).Where("id IN ?", ids).Updates(map[string]any{
			"status": EmailSending, "attempts": gorm.Expr("attempts + 1"), "locked_until": now.Add(lease),
		}).Error
	})
	return claimed, err
}

func (r *emailRepository) MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entities.EmailNotification{}).Where("id = ?", id).Updates(map[string]any{
		"status": EmailSent, "sent_at": sentAt, "locked_until": nil, "last_error": nil,
	}).Error
}

func (r *emailRepository) MarkFailed(ctx context.Context, id uuid.UUID, attempts int, message string, retryAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entities.EmailNotification{}).Where("id = ?", id).Updates(map[string]any{
		"status": EmailPending, "attempts": attempts, "next_attempt_at": retryAt, "locked_until": nil, "last_error": message,
	}).Error
}
