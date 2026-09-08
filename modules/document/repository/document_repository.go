package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDocumentNotFound       = errors.New("one or more documents do not exist")
	ErrDocumentAttachConflict = errors.New("one or more documents are already attached")
)

type DocumentRepository interface {
	Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error)
	GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error)
	ListByPermohonan(ctx context.Context, tx *gorm.DB, permohonanID string, workflowNode *string) ([]entities.Document, error)
	ExistsForWorkflowNode(ctx context.Context, tx *gorm.DB, permohonanID, workflowNode string) (bool, error)
	AttachToWorkflowNode(ctx context.Context, tx *gorm.DB, documentIDs []string, permohonanID, workflowNode, attachedBy string) (int64, error)
}

type documentRepository struct{ db *gorm.DB }

func NewDocumentRepository(db *gorm.DB) DocumentRepository { return &documentRepository{db: db} }

func (r *documentRepository) database(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *documentRepository) Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error) {
	if err := r.database(tx).WithContext(ctx).Create(&document).Error; err != nil {
		return entities.Document{}, err
	}
	return document, nil
}

func (r *documentRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error) {
	var document entities.Document
	if err := r.database(tx).WithContext(ctx).Preload("Evidence").Where("id = ?", id).Take(&document).Error; err != nil {
		return entities.Document{}, err
	}
	return document, nil
}

func (r *documentRepository) ListByPermohonan(ctx context.Context, tx *gorm.DB, permohonanID string, workflowNode *string) ([]entities.Document, error) {
	query := r.database(tx).WithContext(ctx).Model(&entities.Document{}).Where("documents.permohonan_id = ?", permohonanID)
	if workflowNode != nil {
		query = query.Joins("JOIN document_evidence de ON de.document_id = documents.id AND de.permohonan_id = documents.permohonan_id").
			Where("de.workflow_node = ?", *workflowNode)
	}

	var documents []entities.Document
	if err := query.Distinct("documents.*").Preload("Evidence").Order("documents.created_at desc").Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

func (r *documentRepository) ExistsForWorkflowNode(ctx context.Context, tx *gorm.DB, permohonanID, workflowNode string) (bool, error) {
	var count int64
	err := r.database(tx).WithContext(ctx).Model(&entities.DocumentEvidence{}).
		Where("permohonan_id = ? AND workflow_node = ?", permohonanID, workflowNode).Count(&count).Error
	return count > 0, err
}

// AttachToWorkflowNode allows a document already attached to this permohonan to
// be classified under another node. It never moves a document between requests.
func (r *documentRepository) AttachToWorkflowNode(ctx context.Context, tx *gorm.DB, documentIDs []string, permohonanID, workflowNode, attachedBy string) (int64, error) {
	db := r.database(tx).WithContext(ctx)
	permohonanUUID, err := uuid.Parse(permohonanID)
	if err != nil {
		return 0, err
	}
	attachedByUUID, err := uuid.Parse(attachedBy)
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0, len(documentIDs))
	seen := make(map[string]struct{}, len(documentIDs))
	for _, documentID := range documentIDs {
		if _, err := uuid.Parse(documentID); err != nil {
			return 0, err
		}
		if _, ok := seen[documentID]; !ok {
			seen[documentID] = struct{}{}
			ids = append(ids, documentID)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}

	var documents []entities.Document
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "permohonan_id").Where("id IN ?", ids).Find(&documents).Error; err != nil {
		return 0, err
	}
	if len(documents) != len(ids) {
		return 0, ErrDocumentNotFound
	}
	for _, document := range documents {
		if document.PermohonanID != nil && *document.PermohonanID != permohonanUUID {
			return 0, ErrDocumentAttachConflict
		}
	}
	var existingEvidence int64
	if err := db.Model(&entities.DocumentEvidence{}).
		Where("document_id IN ? AND workflow_node = ?", ids, workflowNode).Count(&existingEvidence).Error; err != nil {
		return 0, err
	}
	if existingEvidence > 0 {
		return 0, ErrDocumentAttachConflict
	}

	if err := db.Model(&entities.Document{}).
		Where("id IN ? AND permohonan_id IS NULL", ids).
		Update("permohonan_id", permohonanID).Error; err != nil {
		return 0, err
	}

	evidence := make([]entities.DocumentEvidence, 0, len(ids))
	for _, documentID := range ids {
		id, _ := uuid.Parse(documentID)
		evidence = append(evidence, entities.DocumentEvidence{
			DocumentID: id, PermohonanID: permohonanUUID, WorkflowNode: workflowNode, AttachedBy: attachedByUUID,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&evidence)
	return result.RowsAffected, result.Error
}
