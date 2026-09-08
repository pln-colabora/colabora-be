package repository

import (
	"context"

	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

// SLARuleRepository is a minimal read-only lookup for entities.SLARule — permohonan is
// currently the only consumer, so it lives here rather than as its own top-level module.
type SLARuleRepository interface {
	GetByActivityAndJenis(ctx context.Context, tx *gorm.DB, activityNumber int16, jenisSambungan string) (entities.SLARule, error)
	ListByJenis(ctx context.Context, tx *gorm.DB, jenisSambungan string) ([]entities.SLARule, error)
}

type slaRuleRepository struct {
	db *gorm.DB
}

func NewSLARuleRepository(db *gorm.DB) SLARuleRepository {
	return &slaRuleRepository{
		db: db,
	}
}

func (r *slaRuleRepository) GetByActivityAndJenis(ctx context.Context, tx *gorm.DB, activityNumber int16, jenisSambungan string) (entities.SLARule, error) {
	if tx == nil {
		tx = r.db
	}

	var rule entities.SLARule
	if err := tx.WithContext(ctx).
		Where("activity_number = ? AND jenis_sambungan = ?", activityNumber, jenisSambungan).
		Take(&rule).Error; err != nil {
		return entities.SLARule{}, err
	}

	return rule, nil
}

func (r *slaRuleRepository) ListByJenis(ctx context.Context, tx *gorm.DB, jenisSambungan string) ([]entities.SLARule, error) {
	if tx == nil {
		tx = r.db
	}

	var rules []entities.SLARule
	if err := tx.WithContext(ctx).Where("jenis_sambungan = ?", jenisSambungan).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}
