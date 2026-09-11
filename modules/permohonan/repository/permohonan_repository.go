package repository

import (
	"context"

	"github.com/Caknoooo/go-pagination"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PermohonanRepository interface {
	Create(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, activities []entities.PermohonanActivity, log entities.ActivityLog) (entities.Permohonan, error)
	GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error)
	GetByIdForUpdate(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error)
	SaveWorkflow(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, logs []entities.ActivityLog) error
	List(ctx context.Context, tx *gorm.DB, filter *query.PermohonanFilter) ([]query.Permohonan, int64, error)
	CountByNoPermohonanPrefix(ctx context.Context, tx *gorm.DB, prefix string) (int64, error)
	ListWorkflowNodes(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.PermohonanActivity, error)
	ListActivityLogs(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.ActivityLog, error)
}

func (r *permohonanRepository) GetByIdForUpdate(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error) {
	if tx == nil {
		tx = r.db
	}

	var permohonan entities.Permohonan
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).Take(&permohonan).Error; err != nil {
		return entities.Permohonan{}, err
	}
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("permohonan_id = ?", id).
		Order("stage_number asc, created_at asc").Find(&permohonan.WorkflowNodes).Error; err != nil {
		return entities.Permohonan{}, err
	}
	return permohonan, nil
}

func (r *permohonanRepository) SaveWorkflow(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, logs []entities.ActivityLog) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx)
	if err := db.Model(&entities.Permohonan{}).Where("id = ?", permohonan.ID).Updates(map[string]any{
		"current_stage":         permohonan.CurrentStage,
		"status":                permohonan.Status,
		"kebutuhan_tiang":       permohonan.KebutuhanTiang,
		"nps_delegation_status": permohonan.NpsDelegationStatus,
		"perlu_pdkb":            permohonan.PerluPdkb,
	}).Error; err != nil {
		return err
	}
	for _, node := range permohonan.WorkflowNodes {
		if err := db.Model(&entities.PermohonanActivity{}).
			Where("permohonan_id = ? AND workflow_node = ?", permohonan.ID, node.WorkflowNode).
			Updates(map[string]any{
				"status":       node.Status,
				"payload":      node.Payload,
				"completed_by": node.CompletedBy,
				"completed_at": node.CompletedAt,
			}).Error; err != nil {
			return err
		}
	}
	if len(logs) > 0 {
		if err := db.Create(&logs).Error; err != nil {
			return err
		}
	}
	return nil
}

type permohonanRepository struct {
	db *gorm.DB
}

func NewPermohonanRepository(db *gorm.DB) PermohonanRepository {
	return &permohonanRepository{
		db: db,
	}
}

func (r *permohonanRepository) Create(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, activities []entities.PermohonanActivity, log entities.ActivityLog) (entities.Permohonan, error) {
	if tx == nil {
		tx = r.db
	}

	if err := tx.WithContext(ctx).Create(&permohonan).Error; err != nil {
		return entities.Permohonan{}, err
	}

	for i := range activities {
		activities[i].PermohonanID = permohonan.ID
	}
	if err := tx.WithContext(ctx).Create(&activities).Error; err != nil {
		return entities.Permohonan{}, err
	}

	log.PermohonanID = permohonan.ID
	if err := tx.WithContext(ctx).Create(&log).Error; err != nil {
		return entities.Permohonan{}, err
	}

	return permohonan, nil
}

func (r *permohonanRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error) {
	if tx == nil {
		tx = r.db
	}

	var permohonan entities.Permohonan
	if err := tx.WithContext(ctx).
		Preload("WorkflowNodes", func(db *gorm.DB) *gorm.DB { return db.Order("stage_number asc, created_at asc") }).
		Where("id = ?", id).Take(&permohonan).Error; err != nil {
		return entities.Permohonan{}, err
	}

	return permohonan, nil
}

func (r *permohonanRepository) List(ctx context.Context, tx *gorm.DB, filter *query.PermohonanFilter) ([]query.Permohonan, int64, error) {
	if tx == nil {
		tx = r.db
	}

	results, total, err := pagination.PaginatedQueryWithIncludableAndOptions[query.Permohonan](
		tx.WithContext(ctx),
		filter,
		pagination.PaginatedQueryOptions{Dialect: pagination.PostgreSQL},
	)
	if err != nil {
		return nil, 0, err
	}
	if len(results) == 0 {
		return results, total, nil
	}

	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.ID)
	}
	var nodes []entities.PermohonanActivity
	if err := tx.WithContext(ctx).Where("permohonan_id IN ?", ids).
		Order("stage_number asc, created_at asc").Find(&nodes).Error; err != nil {
		return nil, 0, err
	}
	byPermohonan := make(map[string][]entities.PermohonanActivity, len(results))
	for _, node := range nodes {
		id := node.PermohonanID.String()
		byPermohonan[id] = append(byPermohonan[id], node)
	}
	for i := range results {
		results[i].WorkflowNodes = byPermohonan[results[i].ID]
	}

	return results, total, nil
}

func (r *permohonanRepository) CountByNoPermohonanPrefix(ctx context.Context, tx *gorm.DB, prefix string) (int64, error) {
	if tx == nil {
		tx = r.db
	}

	var count int64
	if err := tx.WithContext(ctx).Model(&entities.Permohonan{}).Where("no_permohonan LIKE ?", prefix+"%").Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *permohonanRepository) ListWorkflowNodes(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.PermohonanActivity, error) {
	if tx == nil {
		tx = r.db
	}
	var nodes []entities.PermohonanActivity
	err := tx.WithContext(ctx).Where("permohonan_id = ?", permohonanID).
		Order("stage_number asc, created_at asc").Find(&nodes).Error
	return nodes, err
}

func (r *permohonanRepository) ListActivityLogs(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.ActivityLog, error) {
	if tx == nil {
		tx = r.db
	}
	var logs []entities.ActivityLog
	err := tx.WithContext(ctx).Preload("ActorUser", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name")
	}).Where("permohonan_id = ?", permohonanID).
		Order("created_at asc").Find(&logs).Error
	return logs, err
}
