package service

import (
	"context"
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
	"gorm.io/gorm"
)

// presignTTL is how long a download link stays valid — short-lived on purpose, since
// evidence documents carry customer PII and the bucket itself is private.
const presignTTL = 15 * time.Minute

type DocumentService interface {
	// Upload stores a raw file standalone — no permohonan/activity context yet, so no
	// ownership check happens here. It's attached later via AttachToActivity.
	Upload(ctx context.Context, userId string, req dto.DocumentUploadRequest) (dto.DocumentResponse, error)
	// AttachToActivity is not yet wired to any HTTP endpoint — built for Phase 4's
	// activity-submission endpoints to call once they exist, referencing documents that
	// were uploaded earlier via Upload.
	AttachToActivity(ctx context.Context, userId, permohonanId string, activityNumber int16, documentIds []string) error
	Download(ctx context.Context, permohonanId, docId string) (string, error)
	List(ctx context.Context, permohonanId string, activityNumber *int16) ([]dto.DocumentResponse, error)
	HasEvidence(ctx context.Context, permohonanId string, activityNumber int16) (bool, error)
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

// AttachToActivity's ownership check is inline rather than middlewares.RequireActivityOwner
// — that middleware bakes its activityNumber in at route-registration time (built for
// Phase 4's one-activity-per-route endpoints), but this is called with a runtime value from
// whichever activity endpoint is submitting. The "permohonan_id IS NULL" guard inside
// AttachToActivity's repository update is what actually prevents hijacking a document
// someone else uploaded — comparing rows-affected here is what surfaces that failure.
func (s *documentService) AttachToActivity(ctx context.Context, userId, permohonanId string, activityNumber int16, documentIds []string) error {
	if len(documentIds) == 0 {
		return nil
	}

	permohonan, err := s.permohonanRepository.GetById(ctx, s.db, permohonanId)
	if err != nil {
		return dto.ErrPermohonanNotFound
	}

	submitter, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.ErrNotActivityOwner
	}

	owns := rbac.OwnsActivity(submitter.Role, submitter.Unit, activityNumber, permohonan.JenisSambungan, permohonan.UlpUnit, permohonan.OwnerFnOverride)
	if !owns {
		return dto.ErrNotActivityOwner
	}

	rowsAffected, err := s.documentRepository.AttachToActivity(ctx, s.db, documentIds, permohonan.ID.String(), activityNumber)
	if err != nil {
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

func (s *documentService) List(ctx context.Context, permohonanId string, activityNumber *int16) ([]dto.DocumentResponse, error) {
	documents, err := s.documentRepository.ListByPermohonan(ctx, s.db, permohonanId, activityNumber)
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
func (s *documentService) HasEvidence(ctx context.Context, permohonanId string, activityNumber int16) (bool, error) {
	return s.documentRepository.ExistsForActivity(ctx, s.db, permohonanId, activityNumber)
}

func toDocumentResponse(d entities.Document) dto.DocumentResponse {
	var permohonanId *string
	if d.PermohonanID != nil {
		id := d.PermohonanID.String()
		permohonanId = &id
	}

	return dto.DocumentResponse{
		ID:             d.ID.String(),
		Type:           d.Type,
		PermohonanID:   permohonanId,
		ActivityNumber: d.ActivityNumber,
		UploadedBy:     d.UploadedBy.String(),
		CreatedAt:      d.CreatedAt.Format(time.RFC3339),
	}
}
