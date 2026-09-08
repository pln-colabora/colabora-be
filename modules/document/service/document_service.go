package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/document/storage"
	permohonanRepository "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

// presignTTL is how long a download link stays valid — short-lived on purpose, since
// evidence documents carry customer PII and the bucket itself is private.
const presignTTL = 15 * time.Minute

type DocumentService interface {
	// Upload stores a raw file standalone — no permohonan/activity context yet, so no
	// ownership check happens here. It's attached later via AttachToWorkflowNode.
	Upload(ctx context.Context, userId string, req dto.DocumentUploadRequest) (dto.DocumentResponse, error)
	// AttachToWorkflowNode is not yet wired to any HTTP endpoint — built for Phase 4's
	// activity-submission endpoints to call once they exist, referencing documents that
	// were uploaded earlier via Upload.
	AttachToWorkflowNode(ctx context.Context, userId, permohonanId, workflowNode string, documentIds []string) error
	Download(ctx context.Context, permohonanId, docId string) (string, error)
	List(ctx context.Context, permohonanId string, workflowNode *string) ([]dto.DocumentResponse, error)
	HasEvidence(ctx context.Context, permohonanId, workflowNode string) (bool, error)
}

type documentService struct {
	documentRepository   repository.DocumentRepository
	permohonanRepository permohonanRepository.PermohonanRepository
	userRepository       userRepository.UserRepository
	storageClient        storage.Client
	db                   *gorm.DB
}

func NewDocumentService(
	documentRepo repository.DocumentRepository,
	permohonanRepo permohonanRepository.PermohonanRepository,
	userRepo userRepository.UserRepository,
	storageClient storage.Client,
	db *gorm.DB,
) DocumentService {
	return &documentService{
		documentRepository:   documentRepo,
		permohonanRepository: permohonanRepo,
		userRepository:       userRepo,
		storageClient:        storageClient,
		db:                   db,
	}
}

func (s *documentService) Upload(ctx context.Context, userId string, req dto.DocumentUploadRequest) (dto.DocumentResponse, error) {
	uploader, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.DocumentResponse{}, err
	}

	file, err := req.File.Open()
	if err != nil {
		return dto.DocumentResponse{}, err
	}
	defer file.Close()

	key := fmt.Sprintf("documents/%s/%s", uuid.New().String(), req.File.Filename)
	contentType := req.File.Header.Get("Content-Type")

	if err := s.storageClient.PutObject(ctx, key, file, req.File.Size, contentType); err != nil {
		return dto.DocumentResponse{}, err
	}

	document := entities.Document{
		Type:       req.Type,
		FilePath:   key,
		UploadedBy: uploader.ID,
	}

	created, err := s.documentRepository.Create(ctx, s.db, document)
	if err != nil {
		return dto.DocumentResponse{}, err
	}

	return toDocumentResponse(created), nil
}

// AttachToWorkflowNode's ownership check is inline: the node is selected at runtime by
// the Phase 4 activity endpoint. The repository refuses to move a file from one
// permohonan to another; Phase 5 will add the aggregate read/access policy.
func (s *documentService) AttachToWorkflowNode(ctx context.Context, userId, permohonanId, workflowNode string, documentIds []string) error {
	if len(documentIds) == 0 {
		return nil
	}
	if _, ok := workflow.Lookup(workflow.Code(workflowNode)); !ok {
		return dto.ErrWorkflowNodeNotFound
	}

	permohonan, err := s.permohonanRepository.GetById(ctx, s.db, permohonanId)
	if err != nil {
		return dto.ErrPermohonanNotFound
	}

	submitter, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.ErrNotWorkflowNodeOwner
	}

	owns := rbac.OwnsWorkflowNode(submitter.Role, submitter.Unit, permohonan.JenisSambungan, permohonan.UlpUnit, workflow.Code(workflowNode))
	if !owns {
		return dto.ErrNotWorkflowNodeOwner
	}

	rowsAffected, err := s.documentRepository.AttachToWorkflowNode(ctx, s.db, documentIds, permohonan.ID.String(), workflowNode, submitter.ID.String())
	if err != nil {
		if errors.Is(err, repository.ErrDocumentNotFound) {
			return dto.ErrDocumentNotFound
		}
		if errors.Is(err, repository.ErrDocumentAttachConflict) {
			return dto.ErrDocumentAlreadyAttached
		}
		return err
	}
	if rowsAffected != int64(len(documentIds)) {
		return dto.ErrDocumentAlreadyAttached
	}

	return nil
}

func (s *documentService) Download(ctx context.Context, permohonanId, docId string) (string, error) {
	document, err := s.documentRepository.GetById(ctx, s.db, docId)
	if err != nil || document.PermohonanID == nil || document.PermohonanID.String() != permohonanId {
		return "", dto.ErrDocumentNotFound
	}

	return s.storageClient.PresignGetObject(ctx, document.FilePath, presignTTL)
}

func (s *documentService) List(ctx context.Context, permohonanId string, workflowNode *string) ([]dto.DocumentResponse, error) {
	documents, err := s.documentRepository.ListByPermohonan(ctx, s.db, permohonanId, workflowNode)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.DocumentResponse, 0, len(documents))
	for _, document := range documents {
		responses = append(responses, toDocumentResponse(document))
	}

	return responses, nil
}

// HasEvidence lets Phase 4 activity endpoints (once built) check required-evidence presence
// per API_SPEC.md's per-endpoint step 2, without reimplementing the query.
func (s *documentService) HasEvidence(ctx context.Context, permohonanId, workflowNode string) (bool, error) {
	return s.documentRepository.ExistsForWorkflowNode(ctx, s.db, permohonanId, workflowNode)
}

func toDocumentResponse(d entities.Document) dto.DocumentResponse {
	var permohonanId *string
	if d.PermohonanID != nil {
		id := d.PermohonanID.String()
		permohonanId = &id
	}

	return dto.DocumentResponse{
		ID:            d.ID.String(),
		Type:          d.Type,
		PermohonanID:  permohonanId,
		WorkflowNodes: evidenceNodes(d.Evidence),
		UploadedBy:    d.UploadedBy.String(),
		CreatedAt:     d.CreatedAt.Format(time.RFC3339),
	}
}

func evidenceNodes(evidence []entities.DocumentEvidence) []string {
	nodes := make([]string, 0, len(evidence))
	for _, item := range evidence {
		nodes = append(nodes, item.WorkflowNode)
	}
	return nodes
}
