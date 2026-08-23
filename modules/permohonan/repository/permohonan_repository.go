package repository

import (
	"context"

	"github.com/Caknoooo/go-pagination"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"gorm.io/gorm"
)

type PermohonanRepository interface {
	Create(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, activities []entities.PermohonanActivity, log entities.ActivityLog) (entities.Permohonan, error)
	GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error)
	List(ctx context.Context, tx *gorm.DB, filter *query.PermohonanFilter) ([]query.Permohonan, int64, error)
	CountByNoPermohonanPrefix(ctx context.Context, tx *gorm.DB, prefix string) (int64, error)
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
	if err := tx.WithContext(ctx).Where("id = ?", id).Take(&permohonan).Error; err != nil {
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
