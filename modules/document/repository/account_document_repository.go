package repository

import (
	"context"

	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

type AccountDocumentRepository interface {
	Create(ctx context.Context, tx *gorm.DB, accountDocument entities.AccountDocument) (entities.AccountDocument, error)
	GetByUserID(ctx context.Context, tx *gorm.DB, userID string) (entities.AccountDocument, error)
}

type accountDocumentRepository struct{ db *gorm.DB }

func NewAccountDocumentRepository(db *gorm.DB) AccountDocumentRepository {
	return &accountDocumentRepository{db: db}
}

func (r *accountDocumentRepository) database(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *accountDocumentRepository) Create(ctx context.Context, tx *gorm.DB, accountDocument entities.AccountDocument) (entities.AccountDocument, error) {
	if err := r.database(tx).WithContext(ctx).Create(&accountDocument).Error; err != nil {
		return entities.AccountDocument{}, err
	}
	return accountDocument, nil
}

func (r *accountDocumentRepository) GetByUserID(ctx context.Context, tx *gorm.DB, userID string) (entities.AccountDocument, error) {
	var accountDocument entities.AccountDocument
	err := r.database(tx).WithContext(ctx).
		Preload("Document").
		Where("user_id = ?", userID).
		Take(&accountDocument).Error
	return accountDocument, err
}
