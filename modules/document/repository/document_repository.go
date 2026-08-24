package repository

import (
	"context"

	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error)
	GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error)
	ListByPermohonan(ctx context.Context, tx *gorm.DB, permohonanId string, activityNumber *int16) ([]entities.Document, error)
	ExistsForActivity(ctx context.Context, tx *gorm.DB, permohonanId string, activityNumber int16) (bool, error)
	// AttachToActivity links the given documents to a permohonan+activity in one atomic
	// update, guarded by "permohonan_id IS NULL" so an already-attached document can never
	// be silently re-attached (hijacked) — callers must compare the returned rows-affected
	// against len(documentIds) and treat a mismatch as failure.
	AttachToActivity(ctx context.Context, tx *gorm.DB, documentIds []string, permohonanId string, activityNumber int16) (int64, error)
}

type documentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{
		db: db,
	}
}

func (r *documentRepository) Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error) {
	if tx == nil {
		tx = r.db
	}

	if err := tx.WithContext(ctx).Create(&document).Error; err != nil {
		return entities.Document{}, err
	}

	return document, nil
}

func (r *documentRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error) {
	if tx == nil {
		tx = r.db
	}

	var document entities.Document
	if err := tx.WithContext(ctx).Where("id = ?", id).Take(&document).Error; err != nil {
		return entities.Document{}, err
	}

	return document, nil
}

func (r *documentRepository) ListByPermohonan(ctx context.Context, tx *gorm.DB, permohonanId string, activityNumber *int16) ([]entities.Document, error) {
	if tx == nil {
		tx = r.db
	}

	query := tx.WithContext(ctx).Where("permohonan_id = ?", permohonanId)
	if activityNumber != nil {
		query = query.Where("activity_number = ?", *activityNumber)
	}

	var documents []entities.Document
	if err := query.Order("created_at desc").Find(&documents).Error; err != nil {
		return nil, err
	}

	return documents, nil
}

func (r *documentRepository) ExistsForActivity(ctx context.Context, tx *gorm.DB, permohonanId string, activityNumber int16) (bool, error) {
	if tx == nil {
		tx = r.db
	}

	var count int64
	if err := tx.WithContext(ctx).Model(&entities.Document{}).
		Where("permohonan_id = ? AND activity_number = ?", permohonanId, activityNumber).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *documentRepository) AttachToActivity(ctx context.Context, tx *gorm.DB, documentIds []string, permohonanId string, activityNumber int16) (int64, error) {
	if tx == nil {
		tx = r.db
	}

	result := tx.WithContext(ctx).Model(&entities.Document{}).
		Where("id IN (?) AND permohonan_id IS NULL", documentIds).
		Updates(map[string]any{
			"permohonan_id":   permohonanId,
			"activity_number": activityNumber,
		})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
