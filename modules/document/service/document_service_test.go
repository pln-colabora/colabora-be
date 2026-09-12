package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/scanning"
	permohonanQuery "github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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
	markSuperseded     bool
	orphans            []entities.Document
	deleted            bool
}

func (f *fakeDocumentRepository) Create(ctx context.Context, tx *gorm.DB, document entities.Document) (entities.Document, error) {
	document.ID = uuid.New()
	f.created = document
	return document, nil
}

func (f *fakeDocumentRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error) {
	return f.byId, f.byIdErr
}
func (f *fakeDocumentRepository) GetByIdForUpdate(ctx context.Context, tx *gorm.DB, id string) (entities.Document, error) {
	return f.byId, f.byIdErr
}
func (f *fakeDocumentRepository) MarkSuperseded(context.Context, *gorm.DB, uuid.UUID, uuid.UUID) (bool, error) {
	return f.markSuperseded, nil
}
func (f *fakeDocumentRepository) ListOrphans(context.Context, *gorm.DB, time.Time, int) ([]entities.Document, error) {
	return f.orphans, nil
}
func (f *fakeDocumentRepository) DeleteUnattached(context.Context, *gorm.DB, uuid.UUID) (bool, error) {
	f.deleted = true
	return true, nil
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
	putCalls     int
	key          string
	size         int64
	contentType  string
	deleteCalls  int
	deleteErr    error
}

func (f *fakeStorageClient) DeleteObject(context.Context, string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeStorageClient) PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	f.putCalls++
	f.key, f.size, f.contentType = key, size, contentType
	return f.putErr
}

func TestUploadRejectsSpoofedContentAndActualOversize(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          error
	}{
		{"spoofed PDF", "<html>not a PDF</html>", dto.ErrInvalidFileType},
		{"empty", "", dto.ErrInvalidFileType},
		{"actual oversize", "%PDF-1.7\n" + strings.Repeat("x", 10<<20), dto.ErrFileTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStorageClient{}
			repo := &fakeDocumentRepository{}
			s := &documentService{documentRepository: repo, userRepository: &fakeUserRepository{user: entities.User{ID: uuid.New()}}, storageClient: store}
			file := newFileHeader(t, "private-customer-name.pdf", "application/pdf", tc.content)
			file.Size = 1 // A service caller cannot bypass the limit by forging metadata.
			_, err := s.Upload(context.Background(), uuid.NewString(), dto.DocumentUploadRequest{File: file, Type: "evidence"})
			require.ErrorIs(t, err, tc.want)
			require.Zero(t, store.putCalls)
			require.Equal(t, uuid.Nil, repo.created.ID)
		})
	}
}

func TestUploadUsesOpaqueKeyAndMeasuredSize(t *testing.T) {
	store := &fakeStorageClient{}
	s := &documentService{documentRepository: &fakeDocumentRepository{}, userRepository: &fakeUserRepository{user: entities.User{ID: uuid.New()}}, storageClient: store}
	content := "%PDF-1.7\nsynthetic"
	file := newFileHeader(t, "private-customer-name.pdf", "application/pdf", content)
	file.Size = 1
	_, err := s.Upload(context.Background(), uuid.NewString(), dto.DocumentUploadRequest{File: file, Type: "evidence"})
	require.NoError(t, err)
	require.NotContains(t, store.key, file.Filename)
	_, err = uuid.Parse(strings.TrimPrefix(store.key, "documents/"))
	require.NoError(t, err)
	require.EqualValues(t, len(content), store.size)
	require.Equal(t, "application/pdf", store.contentType)
	require.Equal(t, "private-customer-name.pdf", s.documentRepository.(*fakeDocumentRepository).created.OriginalFilename)
	require.Len(t, s.documentRepository.(*fakeDocumentRepository).created.ChecksumSHA256, 64)
}

type fakeScanner struct {
	status string
	err    error
}

func (f fakeScanner) Scan(context.Context, []byte) (string, error) { return f.status, f.err }

func TestUploadScannerIsFailClosedWhenConfigured(t *testing.T) {
	for _, tc := range []struct {
		name    string
		scanner scanning.Scanner
		want    error
	}{
		{"infected", fakeScanner{status: scanning.StatusInfected}, dto.ErrMalwareDetected},
		{"scanner failure", fakeScanner{err: errors.New("synthetic")}, dto.ErrScanFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStorageClient{}
			s := &documentService{documentRepository: &fakeDocumentRepository{}, userRepository: &fakeUserRepository{user: entities.User{ID: uuid.New()}}, storageClient: store, scanner: tc.scanner}
			_, err := s.Upload(context.Background(), uuid.NewString(), dto.DocumentUploadRequest{File: newFileHeader(t, "evidence.pdf", "application/pdf", "%PDF-1.7\nsynthetic"), Type: "evidence"})
			require.ErrorIs(t, err, tc.want)
			require.Zero(t, store.putCalls)
		})
	}
}

func TestUploadRevisionOnlySupersedesOwnUnattachedUpload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	uploader := uuid.New()
	previous := entities.Document{ID: uuid.New(), Type: "evidence", UploadedBy: uploader, Revision: 1}
	repo := &fakeDocumentRepository{byId: previous, markSuperseded: true}
	s := &documentService{documentRepository: repo, userRepository: &fakeUserRepository{user: entities.User{ID: uploader}}, storageClient: &fakeStorageClient{}, scanner: fakeScanner{status: scanning.StatusClean}, db: db}
	result, err := s.Upload(context.Background(), uploader.String(), dto.DocumentUploadRequest{File: newFileHeader(t, "replacement.pdf", "application/pdf", "%PDF-1.7\nreplacement"), Type: "evidence", SupersedesDocumentID: previous.ID.String()})
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Revision)
	require.Equal(t, previous.ID.String(), *result.SupersedesID)
	require.NotNil(t, result.ScanCheckedAt)

	previous.PermohonanID = ptrUUID(uuid.New())
	repo.byId = previous
	_, err = s.Upload(context.Background(), uploader.String(), dto.DocumentUploadRequest{File: newFileHeader(t, "replacement.pdf", "application/pdf", "%PDF-1.7\nreplacement"), Type: "evidence", SupersedesDocumentID: previous.ID.String()})
	require.ErrorIs(t, err, dto.ErrInvalidRevision)
}

func TestCleanupOrphansDeletesStorageBeforeMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	old := entities.Document{ID: uuid.New(), FilePath: "documents/opaque", Timestamp: entities.Timestamp{CreatedAt: time.Now().Add(-48 * time.Hour)}}
	repo := &fakeDocumentRepository{orphans: []entities.Document{old}, byId: old}
	store := &fakeStorageClient{}
	s := &documentService{documentRepository: repo, storageClient: store, db: db}
	deleted, err := s.CleanupOrphans(context.Background(), time.Now().Add(-24*time.Hour), 100)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.Equal(t, 1, store.deleteCalls)
	require.True(t, repo.deleted)
}

func TestPreviewAllowsOnlyUploaderForUnattachedDocument(t *testing.T) {
	uploader := uuid.New()
	repo := &fakeDocumentRepository{byId: entities.Document{
		ID: uploader, FilePath: "documents/preview", OriginalFilename: "evidence.pdf",
		MimeType: "application/pdf", UploadedBy: uploader, ScanStatus: scanning.StatusNotScanned,
	}}
	store := &fakeStorageClient{presignedURL: "https://signed.example/preview"}
	s := &documentService{documentRepository: repo, storageClient: store}

	result, err := s.Preview(context.Background(), uploader.String(), uploader.String())
	require.NoError(t, err)
	require.Equal(t, store.presignedURL, result.URL)
	require.Equal(t, "application/pdf", result.MimeType)

	_, err = s.Preview(context.Background(), uuid.NewString(), uploader.String())
	require.ErrorIs(t, err, dto.ErrDocumentNotFound)
}

func ptrUUID(value uuid.UUID) *uuid.UUID { return &value }

func (f *fakeStorageClient) PresignGetObject(ctx context.Context, key string, ttl time.Duration, contentDisposition string) (string, error) {
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
		File: newFileHeader(t, "evidence.pdf", "application/pdf", "%PDF-1.7\nsynthetic evidence"),
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

	t.Run("infected document is unavailable", func(t *testing.T) {
		infected := document
		infected.ScanStatus = scanning.StatusInfected
		s := &documentService{documentRepository: &fakeDocumentRepository{byId: infected}, storageClient: &fakeStorageClient{}}
		_, err := s.Download(context.Background(), permohonanId.String(), docId.String())
		assert.ErrorIs(t, err, dto.ErrDocumentUnavailable)
	})
}

func TestDocumentService_HasEvidence(t *testing.T) {
	s := &documentService{documentRepository: &fakeDocumentRepository{existsOK: true}}

	ok, err := s.HasEvidence(context.Background(), uuid.NewString(), "survei")
	require.NoError(t, err)
	assert.True(t, ok)
}
