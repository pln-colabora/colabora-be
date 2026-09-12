package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/document/scanning"
	"github.com/pln-colabora/colabora-be/modules/document/storage"
	"github.com/pln-colabora/colabora-be/modules/document/validation"
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
	// Activity-submission endpoints call AttachToWorkflowNode in their transaction,
	// referencing documents uploaded earlier via Upload.
	AttachToWorkflowNode(ctx context.Context, userId, permohonanId, workflowNode string, documentIds []string) error
	Download(ctx context.Context, permohonanId, docId string) (string, error)
	Preview(ctx context.Context, userID, docID string) (dto.DocumentPreviewResponse, error)
	List(ctx context.Context, permohonanId string, workflowNode *string) ([]dto.DocumentResponse, error)
	HasEvidence(ctx context.Context, permohonanId, workflowNode string) (bool, error)
	CleanupOrphans(ctx context.Context, before time.Time, limit int) (int, error)
}

type documentService struct {
	documentRepository   repository.DocumentLifecycleRepository
	permohonanRepository permohonanRepository.PermohonanRepository
	userRepository       userRepository.UserRepository
	storageClient        storage.Client
	scanner              scanning.Scanner
	db                   *gorm.DB
}

func NewDocumentService(
	documentRepo repository.DocumentLifecycleRepository,
	permohonanRepo permohonanRepository.PermohonanRepository,
	userRepo userRepository.UserRepository,
	storageClient storage.Client,
	scanner scanning.Scanner,
	db *gorm.DB,
) DocumentService {
	return &documentService{
		documentRepository:   documentRepo,
		permohonanRepository: permohonanRepo,
		userRepository:       userRepo,
		storageClient:        storageClient,
		scanner:              scanner,
		db:                   db,
	}
}

func (s *documentService) Upload(ctx context.Context, userId string, req dto.DocumentUploadRequest) (dto.DocumentResponse, error) {
	if err := validation.NewDocumentValidation().ValidateDocumentUploadRequest(req); err != nil {
		return dto.DocumentResponse{}, err
	}
	uploader, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.DocumentResponse{}, err
	}

	file, err := req.File.Open()
	if err != nil {
		return dto.DocumentResponse{}, err
	}
	defer file.Close()

	// Bound the actual bytes even when a caller supplies an incorrect FileHeader.Size.
	content, err := io.ReadAll(io.LimitReader(file, validation.MaxUploadSizeBytes+1))
	if err != nil {
		return dto.DocumentResponse{}, dto.ErrInvalidFileType
	}
	if len(content) > validation.MaxUploadSizeBytes {
		return dto.DocumentResponse{}, dto.ErrFileTooLarge
	}
	contentType := http.DetectContentType(content)
	if len(content) == 0 || contentType != req.File.Header.Get("Content-Type") {
		return dto.DocumentResponse{}, dto.ErrInvalidFileType
	}
	scanner := s.scanner
	if scanner == nil {
		scanner = scanning.DisabledScanner{}
	}
	scanStatus, err := scanner.Scan(ctx, content)
	if err != nil {
		return dto.DocumentResponse{}, dto.ErrScanFailed
	}
	if scanStatus == scanning.StatusInfected {
		return dto.DocumentResponse{}, dto.ErrMalwareDetected
	}
	if scanStatus != scanning.StatusClean && scanStatus != scanning.StatusNotScanned {
		return dto.DocumentResponse{}, dto.ErrScanFailed
	}
	key := fmt.Sprintf("documents/%s", uuid.New().String())

	if err := s.storageClient.PutObject(ctx, key, bytes.NewReader(content), int64(len(content)), contentType); err != nil {
		return dto.DocumentResponse{}, err
	}

	checksum := sha256.Sum256(content)
	document := entities.Document{
		Type: strings.TrimSpace(req.Type), FilePath: key, OriginalFilename: strings.TrimSpace(filepath.Base(req.File.Filename)),
		MimeType: contentType, SizeBytes: int64(len(content)), ChecksumSHA256: fmt.Sprintf("%x", checksum),
		Source: "uploaded", Classification: "restricted", ScanStatus: scanStatus, Revision: 1,
		UploadedBy: uploader.ID,
	}
	if scanStatus == scanning.StatusClean {
		now := time.Now()
		document.ScanCheckedAt = &now
	}

	var created entities.Document
	if req.SupersedesDocumentID == "" {
		created, err = s.documentRepository.Create(ctx, s.db, document)
	} else {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			previous, txErr := s.documentRepository.GetByIdForUpdate(ctx, tx, req.SupersedesDocumentID)
			if txErr != nil {
				if errors.Is(txErr, gorm.ErrRecordNotFound) {
					return dto.ErrInvalidRevision
				}
				return txErr
			}
			if previous.UploadedBy != uploader.ID || previous.PermohonanID != nil || len(previous.Evidence) != 0 || previous.SupersededByID != nil || previous.Type != document.Type || previous.Revision >= 32767 {
				return dto.ErrInvalidRevision
			}
			document.Revision = previous.Revision + 1
			if document.Revision < 2 {
				document.Revision = 2
			}
			document.SupersedesID = &previous.ID
			created, txErr = s.documentRepository.Create(ctx, tx, document)
			if txErr != nil {
				return txErr
			}
			updated, txErr := s.documentRepository.MarkSuperseded(ctx, tx, previous.ID, created.ID)
			if txErr != nil {
				return txErr
			}
			if !updated {
				return dto.ErrInvalidRevision
			}
			return nil
		})
	}
	if err != nil {
		// The opaque upload is not referenced if persistence fails. S3 deletion is
		// idempotent and is attempted immediately as compensation.
		_ = s.storageClient.DeleteObject(ctx, key)
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
	if document.ScanStatus == scanning.StatusInfected {
		return "", dto.ErrDocumentUnavailable
	}

	return s.storageClient.PresignGetObject(ctx, document.FilePath, presignTTL, "attachment")
}

func (s *documentService) Preview(ctx context.Context, userID, docID string) (dto.DocumentPreviewResponse, error) {
	document, err := s.documentRepository.GetById(ctx, s.db, docID)
	if err != nil {
		return dto.DocumentPreviewResponse{}, dto.ErrDocumentNotFound
	}
	if document.SupersededByID != nil || document.ScanStatus == scanning.StatusInfected {
		return dto.DocumentPreviewResponse{}, dto.ErrDocumentUnavailable
	}

	if document.PermohonanID == nil {
		if document.UploadedBy.String() != userID {
			return dto.DocumentPreviewResponse{}, dto.ErrDocumentNotFound
		}
	} else if !s.canReadPermohonan(ctx, userID, document.PermohonanID.String()) {
		return dto.DocumentPreviewResponse{}, dto.ErrDocumentNotFound
	}

	const previewTTL = 15 * time.Minute
	url, err := s.storageClient.PresignGetObject(ctx, document.FilePath, previewTTL, "inline")
	if err != nil {
		return dto.DocumentPreviewResponse{}, err
	}
	return dto.DocumentPreviewResponse{
		URL: url, MimeType: document.MimeType, Filename: document.OriginalFilename,
		ExpiresAt: time.Now().Add(previewTTL).UTC().Format(time.RFC3339),
	}, nil
}

func (s *documentService) canReadPermohonan(ctx context.Context, userID, permohonanID string) bool {
	var actor entities.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).Take(&actor).Error; err != nil {
		return false
	}
	var count int64
	err := rbac.ApplyReadScope(s.db.WithContext(ctx).Model(&entities.Permohonan{}), actor.Role, actor.Unit, actor.ID.String()).
		Where("permohonan.id = ?", permohonanID).Count(&count).Error
	return err == nil && count == 1
}

func (s *documentService) CleanupOrphans(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	documents, err := s.documentRepository.ListOrphans(ctx, s.db, before, limit)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, candidate := range documents {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			document, txErr := s.documentRepository.GetByIdForUpdate(ctx, tx, candidate.ID.String())
			if txErr != nil {
				return txErr
			}
			if document.PermohonanID != nil || !document.CreatedAt.Before(before) {
				return nil
			}
			if txErr = s.storageClient.DeleteObject(ctx, document.FilePath); txErr != nil {
				return txErr
			}
			removed, txErr := s.documentRepository.DeleteUnattached(ctx, tx, document.ID)
			if txErr != nil {
				return txErr
			}
			if removed {
				deleted++
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return deleted, err
		}
	}
	return deleted, nil
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

	var supersedesID, supersededByID *string
	if d.SupersedesID != nil {
		value := d.SupersedesID.String()
		supersedesID = &value
	}
	if d.SupersededByID != nil {
		value := d.SupersededByID.String()
		supersededByID = &value
	}
	var scanCheckedAt *string
	if d.ScanCheckedAt != nil {
		value := d.ScanCheckedAt.Format(time.RFC3339)
		scanCheckedAt = &value
	}
	return dto.DocumentResponse{
		ID:               d.ID.String(),
		Type:             d.Type,
		OriginalFilename: d.OriginalFilename, MimeType: d.MimeType, SizeBytes: d.SizeBytes,
		ChecksumSHA256: d.ChecksumSHA256, Source: d.Source, Classification: d.Classification,
		ScanStatus: d.ScanStatus, ScanCheckedAt: scanCheckedAt, Revision: d.Revision, SupersedesID: supersedesID, SupersededByID: supersededByID,
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
