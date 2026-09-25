package repository

import (
	"context"

	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

type TariffPowerRepository interface {
	List(ctx context.Context, tx *gorm.DB, jenisSambungan string) ([]entities.TariffPowerOption, error)
	IsAllowed(ctx context.Context, tx *gorm.DB, tarif, jenisSambungan string, daya int64) (bool, error)
}

type tariffPowerRepository struct{ db *gorm.DB }

func NewTariffPowerRepository(db *gorm.DB) TariffPowerRepository {
	return &tariffPowerRepository{db: db}
}

func (r *tariffPowerRepository) database(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *tariffPowerRepository) List(ctx context.Context, tx *gorm.DB, jenisSambungan string) ([]entities.TariffPowerOption, error) {
	var options []entities.TariffPowerOption
	db := r.database(tx).WithContext(ctx).Where("active = ?", true)
	if jenisSambungan != "" {
		db = db.Where("jenis_sambungan = ?", jenisSambungan)
	}
	err := db.Order("tarif asc, daya_min asc, daya_max asc").Find(&options).Error
	return options, err
}

func (r *tariffPowerRepository) IsAllowed(ctx context.Context, tx *gorm.DB, tarif, jenisSambungan string, daya int64) (bool, error) {
	var count int64
	err := r.database(tx).WithContext(ctx).Model(&entities.TariffPowerOption{}).
		Where("tarif = ? AND jenis_sambungan = ? AND active = ?", tarif, jenisSambungan, true).
		Where("daya_min <= ?", daya).
		Where("daya_max IS NULL OR daya_max >= ?", daya).
		Count(&count).Error
	return count > 0, err
}
