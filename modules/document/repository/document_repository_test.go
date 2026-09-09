package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAttachToWorkflowNodeValidatesAndClassifiesEvidence(t *testing.T) {
	db := documentRepositoryDB(t)
	repo := NewDocumentRepository(db)
	actorID := uuid.NewString()
	permohonanID := uuid.NewString()
	document := entities.Document{ID: uuid.New(), Type: "evidence", FilePath: "private/a.pdf", UploadedBy: uuid.MustParse(actorID)}
	require.NoError(t, db.Create(&document).Error)

	rows, err := repo.AttachToWorkflowNode(context.Background(), db, []string{document.ID.String()}, permohonanID, "rab_kko_kkf", actorID)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)

	rows, err = repo.AttachToWorkflowNode(context.Background(), db, []string{document.ID.String()}, permohonanID, "kebutuhan_tiang", actorID)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows, "same-request evidence may classify the bundled decision node")

	_, err = repo.AttachToWorkflowNode(context.Background(), db, []string{document.ID.String()}, permohonanID, "rab_kko_kkf", actorID)
	require.ErrorIs(t, err, ErrDocumentAttachConflict)

	_, err = repo.AttachToWorkflowNode(context.Background(), db, []string{uuid.NewString()}, permohonanID, "survei", actorID)
	require.ErrorIs(t, err, ErrDocumentNotFound)
}

func TestAttachToWorkflowNodeRejectsDocumentFromAnotherPermohonan(t *testing.T) {
	db := documentRepositoryDB(t)
	repo := NewDocumentRepository(db)
	otherPermohonanID := uuid.New()
	document := entities.Document{
		ID: uuid.New(), Type: "evidence", FilePath: "private/b.pdf", UploadedBy: uuid.New(), PermohonanID: &otherPermohonanID,
	}
	require.NoError(t, db.Create(&document).Error)

	_, err := repo.AttachToWorkflowNode(
		context.Background(), db, []string{document.ID.String()}, uuid.NewString(), "survei", uuid.NewString(),
	)
	require.ErrorIs(t, err, ErrDocumentAttachConflict)
}

func TestAttachRejectsAnotherUsersUnattachedUpload(t *testing.T) {
	db := documentRepositoryDB(t)
	doc := entities.Document{ID: uuid.New(), Type: "evidence", FilePath: "private/synthetic.pdf", UploadedBy: uuid.New()}
	require.NoError(t, db.Create(&doc).Error)
	_, err := NewDocumentRepository(db).AttachToWorkflowNode(context.Background(), db, []string{doc.ID.String()}, uuid.NewString(), "survei", uuid.NewString())
	require.ErrorIs(t, err, ErrDocumentNotFound)
	require.NoError(t, db.First(&doc, "id = ?", doc.ID).Error)
	require.Nil(t, doc.PermohonanID)
	var count int64
	require.NoError(t, db.Model(&entities.DocumentEvidence{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestAttachRejectsSupersededUpload(t *testing.T) {
	db := documentRepositoryDB(t)
	actor := uuid.New()
	replacement := uuid.New()
	doc := entities.Document{ID: uuid.New(), Type: "evidence", FilePath: "private/old", UploadedBy: actor, SupersededByID: &replacement}
	require.NoError(t, db.Create(&doc).Error)
	_, err := NewDocumentRepository(db).AttachToWorkflowNode(context.Background(), db, []string{doc.ID.String()}, uuid.NewString(), "survei", actor.String())
	require.ErrorIs(t, err, ErrDocumentNotFound)
}

func documentRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE documents (id TEXT PRIMARY KEY, type TEXT, file_path TEXT, original_filename TEXT, mime_type TEXT, size_bytes INTEGER, checksum_sha256 TEXT, source TEXT, classification TEXT, scan_status TEXT, scan_checked_at DATETIME, revision INTEGER, supersedes_id TEXT, superseded_by_id TEXT, uploaded_by TEXT, permohonan_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE document_evidence (id TEXT PRIMARY KEY, document_id TEXT, permohonan_id TEXT, workflow_node TEXT, attached_by TEXT, created_at DATETIME, updated_at DATETIME, UNIQUE(document_id, workflow_node))`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}
