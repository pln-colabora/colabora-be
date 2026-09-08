package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	permohonanQuery "github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- fakes -------------------------------------------------------------

type fakeDocumentRepository struct {
	created            entities.Document
	byId               entities.Document
	byIdErr            error
	existsOK           bool
	attachRowsAffected int64
	attachErr          error
}

func (f *fakeDocumentRepository) Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error) {
	document.ID = uuid.New()
	f.created = document
	return document, nil
}

func (f *fakeDocumentRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error) {
	return f.byId, f.byIdErr
}

func (f *fakeDocumentRepository) ListByPermohonan(ctx context.Context, tx *gorm.DB, permohonanId string, workflowNode *string) ([]entities.Document, error) {
	return []entities.Document{f.byId}, nil
}

func (f *fakeDocumentRepository) ExistsForWorkflowNode(ctx context.Context, tx *gorm.DB, permohonanId, workflowNode string) (bool, error) {
	return f.existsOK, nil
}

func (f *fakeDocumentRepository) AttachToWorkflowNode(ctx context.Context, tx *gorm.DB, documentIds []string, permohonanId, workflowNode, attachedBy string) (int64, error) {
	if f.attachErr != nil {
		return 0, f.attachErr
	}
	return f.attachRowsAffected, nil
}

type fakePermohonanRepository struct {
	permohonan entities.Permohonan
	err        error
}

func (f *fakePermohonanRepository) Create(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, activities []entities.PermohonanActivity, log entities.ActivityLog) (entities.Permohonan, error) {
	return entities.Permohonan{}, nil
}
func (f *fakePermohonanRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error) {
	return f.permohonan, f.err
}
func (f *fakePermohonanRepository) GetByIdForUpdate(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error) {
	return f.permohonan, f.err
}
func (f *fakePermohonanRepository) SaveWorkflow(context.Context, *gorm.DB, entities.Permohonan, []entities.ActivityLog) error {
	return nil
}
func (f *fakePermohonanRepository) List(ctx context.Context, tx *gorm.DB, filter *permohonanQuery.PermohonanFilter) ([]permohonanQuery.Permohonan, int64, error) {
	return nil, 0, nil
}
func (f *fakePermohonanRepository) CountByNoPermohonanPrefix(ctx context.Context, tx *gorm.DB, prefix string) (int64, error) {
	return 0, nil
}
func (f *fakePermohonanRepository) ListWorkflowNodes(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.PermohonanActivity, error) {
	return nil, nil
}
func (f *fakePermohonanRepository) ListActivityLogs(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.ActivityLog, error) {
	return nil, nil
}

type fakeUserRepository struct {
	user entities.User
	err  error
}

func (f *fakeUserRepository) Register(ctx context.Context, tx *gorm.DB, user entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) GetUserById(ctx context.Context, tx *gorm.DB, userId string) (entities.User, error) {
	return f.user, f.err
}
func (f *fakeUserRepository) GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) CheckEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, bool, error) {
	return entities.User{}, false, nil
}
func (f *fakeUserRepository) Update(ctx context.Context, tx *gorm.DB, user entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) Delete(ctx context.Context, tx *gorm.DB, userId string) error {
	return nil
}
func (f *fakeUserRepository) ExistsByUnitAndRoles(ctx context.Context, tx *gorm.DB, unit string, roles []string) (bool, error) {
	return true, nil
}

type fakeStorageClient struct {
	putErr       error
	presignedURL string
}

func (f *fakeStorageClient) PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	return f.putErr
}

func (f *fakeStorageClient) PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return f.presignedURL, nil
}

// --- helpers -------------------------------------------------------------

func newFileHeader(t *testing.T, filename, contentType, content string) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)},
		"Content-Type":        []string{contentType},
	})
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	require.NoError(t, err)
	t.Cleanup(func() { _ = form.RemoveAll() })

	return form.File["file"][0]
}

// --- tests -------------------------------------------------------------

func TestDocumentService_Upload(t *testing.T) {
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s := &documentService{
		documentRepository: &fakeDocumentRepository{},
		userRepository:     &fakeUserRepository{user: user},
		storageClient:      &fakeStorageClient{},
	}

	req := dto.DocumentUploadRequest{
		File: newFileHeader(t, "evidence.pdf", "application/pdf", "hello"),
		Type: "evidence",
	}

	result, err := s.Upload(context.Background(), user.ID.String(), req)
	require.NoError(t, err)
	assert.Equal(t, "evidence", result.Type)
	assert.Nil(t, result.PermohonanID)
	assert.Empty(t, result.WorkflowNodes)
}

func TestDocumentService_AttachToWorkflowNode(t *testing.T) {
	permohonan := entities.Permohonan{
		ID:             uuid.New(),
		JenisSambungan: rbac.JenisSambunganJTR,
		UlpUnit:        "ULP Taman",
	}
	documentIds := []string{uuid.NewString(), uuid.NewString()}

	t.Run("owning role attaches successfully", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
		s := &documentService{
			documentRepository:   &fakeDocumentRepository{attachRowsAffected: 2},
			permohonanRepository: &fakePermohonanRepository{permohonan: permohonan},
			userRepository:       &fakeUserRepository{user: user},
		}

		err := s.AttachToWorkflowNode(context.Background(), user.ID.String(), permohonan.ID.String(), "survei", documentIds)
		require.NoError(t, err)
	})

	t.Run("non-owning role is rejected", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RolePelayananPelanggan, Unit: "ULP Taman"}
		s := &documentService{
			documentRepository:   &fakeDocumentRepository{attachRowsAffected: 2},
			permohonanRepository: &fakePermohonanRepository{permohonan: permohonan},
			userRepository:       &fakeUserRepository{user: user},
		}

		err := s.AttachToWorkflowNode(context.Background(), user.ID.String(), permohonan.ID.String(), "survei", documentIds)
		assert.ErrorIs(t, err, dto.ErrNotWorkflowNodeOwner)
	})

	t.Run("already-attached document rejects the whole batch", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
		s := &documentService{
			// only 1 of 2 documents matched the "permohonan_id IS NULL" guard
			documentRepository:   &fakeDocumentRepository{attachRowsAffected: 1},
			permohonanRepository: &fakePermohonanRepository{permohonan: permohonan},
			userRepository:       &fakeUserRepository{user: user},
		}

		err := s.AttachToWorkflowNode(context.Background(), user.ID.String(), permohonan.ID.String(), "survei", documentIds)
		assert.ErrorIs(t, err, dto.ErrDocumentAlreadyAttached)
	})

	t.Run("unknown permohonan", func(t *testing.T) {
		s := &documentService{
			documentRepository:   &fakeDocumentRepository{},
			permohonanRepository: &fakePermohonanRepository{err: gorm.ErrRecordNotFound},
			userRepository:       &fakeUserRepository{},
		}

		err := s.AttachToWorkflowNode(context.Background(), uuid.NewString(), uuid.NewString(), "survei", documentIds)
		assert.ErrorIs(t, err, dto.ErrPermohonanNotFound)
	})
}

func TestDocumentService_Download(t *testing.T) {
	permohonanId := uuid.New()
	docId := uuid.New()
	document := entities.Document{ID: docId, FilePath: "documents/x/key.pdf", PermohonanID: &permohonanId}

	t.Run("returns a presigned url", func(t *testing.T) {
		s := &documentService{
			documentRepository: &fakeDocumentRepository{byId: document},
			storageClient:      &fakeStorageClient{presignedURL: "https://garage.local/presigned"},
		}

		url, err := s.Download(context.Background(), permohonanId.String(), docId.String())
		require.NoError(t, err)
		assert.Equal(t, "https://garage.local/presigned", url)
	})

	t.Run("unknown document", func(t *testing.T) {
		s := &documentService{
			documentRepository: &fakeDocumentRepository{byIdErr: gorm.ErrRecordNotFound},
			storageClient:      &fakeStorageClient{},
		}

		_, err := s.Download(context.Background(), permohonanId.String(), uuid.NewString())
		assert.ErrorIs(t, err, dto.ErrDocumentNotFound)
	})

	t.Run("document belongs to a different permohonan", func(t *testing.T) {
		s := &documentService{
			documentRepository: &fakeDocumentRepository{byId: document},
			storageClient:      &fakeStorageClient{},
		}

		_, err := s.Download(context.Background(), uuid.NewString(), docId.String())
		assert.ErrorIs(t, err, dto.ErrDocumentNotFound)
	})

	t.Run("unattached document", func(t *testing.T) {
		s := &documentService{
			documentRepository: &fakeDocumentRepository{byId: entities.Document{ID: docId}},
			storageClient:      &fakeStorageClient{},
		}

		_, err := s.Download(context.Background(), permohonanId.String(), docId.String())
		assert.ErrorIs(t, err, dto.ErrDocumentNotFound)
	})
}

func TestDocumentService_HasEvidence(t *testing.T) {
	s := &documentService{documentRepository: &fakeDocumentRepository{existsOK: true}}

	ok, err := s.HasEvidence(context.Background(), uuid.NewString(), "survei")
	require.NoError(t, err)
	assert.True(t, ok)
}
