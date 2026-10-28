package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowChangesRepository struct{}

func (WorkflowChangesRepository) ListSchedules(ctx context.Context, db *gorm.DB, from, to time.Time) ([]entities.PDKBSchedule, error) {
	rows := []entities.PDKBSchedule{}
	err := db.WithContext(ctx).Where("schedule_date >= ? AND schedule_date <= ?", from, to).Order("schedule_date").Find(&rows).Error
	return rows, err
}

func scheduleError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) || (err != nil && (strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "UNIQUE constraint failed"))) {
		return dto.ErrScheduleConflict
	}
	return err
}

func (WorkflowChangesRepository) SaveSchedule(ctx context.Context, db *gorm.DB, item *entities.PDKBSchedule, create bool) error {
	if create {
		return scheduleError(db.WithContext(ctx).Create(item).Error)
	}
	var old entities.PDKBSchedule
	if err := db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", item.ID).Take(&old).Error; err != nil {
		return err
	}
	if old.PermohonanID != nil {
		return dto.ErrScheduleLocked
	}
	return scheduleError(db.WithContext(ctx).Model(&old).Updates(map[string]any{"schedule_date": item.ScheduleDate, "title": item.Title, "updated_by": item.UpdatedBy}).Error)
}

func (WorkflowChangesRepository) DeleteSchedule(ctx context.Context, db *gorm.DB, id string) error {
	var item entities.PDKBSchedule
	if err := db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&item).Error; err != nil {
		return err
	}
	if item.PermohonanID != nil {
		return dto.ErrScheduleLocked
	}
	return db.WithContext(ctx).Delete(&item).Error
}

func (WorkflowChangesRepository) Revisions(ctx context.Context, db *gorm.DB, id, code string) ([]entities.ActivityRevision, error) {
	rows := []entities.ActivityRevision{}
	err := db.WithContext(ctx).Where("permohonan_id = ? AND workflow_node = ?", id, code).Order("revision").Find(&rows).Error
	return rows, err
}

func (WorkflowChangesRepository) SaveRevision(ctx context.Context, db *gorm.DB, old entities.ActivityRevision, payload string, next int) error {
	if err := db.WithContext(ctx).Create(&old).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&entities.PermohonanActivity{}).Where("permohonan_id = ? AND workflow_node = ?", old.PermohonanID, old.WorkflowNode).Updates(map[string]any{"payload": payload, "revision": next}).Error
}

func (WorkflowChangesRepository) Evidence(ctx context.Context, db *gorm.DB, id, code string) ([]entities.DocumentEvidence, error) {
	rows := []entities.DocumentEvidence{}
	err := db.WithContext(ctx).Where("permohonan_id = ? AND workflow_node = ?", id, code).Find(&rows).Error
	return rows, err
}

// Generated WO files are immutable; only uploaded attachments participate in revisions.
func (WorkflowChangesRepository) SelectEvidence(ctx context.Context, db *gorm.DB, id, code string, selected []string) error {
	q := db.WithContext(ctx).Model(&entities.DocumentEvidence{}).Where("permohonan_id = ? AND workflow_node = ?", id, code).
		Where("document_id IN (SELECT id FROM documents WHERE source = 'uploaded')")
	if err := q.Update("is_current", false).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&entities.DocumentEvidence{}).Where("permohonan_id = ? AND workflow_node = ? AND document_id IN ?", id, code, selected).Update("is_current", true).Error
}

func (WorkflowChangesRepository) LogRevision(ctx context.Context, db *gorm.DB, id, actor uuid.UUID, code string, detail string) error {
	return db.WithContext(ctx).Create(&entities.ActivityLog{PermohonanID: id, Actor: actor, WorkflowNode: &code, Action: "node_revised", Detail: &detail}).Error
}
