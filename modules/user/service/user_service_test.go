package service

import (
	"context"
	"testing"
	"time"

	"github.com/Caknoooo/go-pagination"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/user/query"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListAccountsIncludesOnlyRegistrationDocumentForItsAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, email TEXT NOT NULL, telp_number TEXT,
		password TEXT NOT NULL, role TEXT NOT NULL, unit TEXT, image_url TEXT,
		is_verified BOOLEAN, created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE documents (
		id TEXT PRIMARY KEY, type TEXT NOT NULL, file_path TEXT NOT NULL,
		original_filename TEXT NOT NULL, mime_type TEXT NOT NULL, size_bytes INTEGER NOT NULL,
		checksum_sha256 TEXT NOT NULL, source TEXT NOT NULL, classification TEXT NOT NULL,
		scan_status TEXT NOT NULL, scan_checked_at DATETIME, revision INTEGER NOT NULL,
		supersedes_id TEXT, superseded_by_id TEXT, uploaded_by TEXT NOT NULL,
		permohonan_id TEXT, created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE account_documents (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, document_id TEXT NOT NULL,
		document_type TEXT NOT NULL, created_at DATETIME, updated_at DATETIME
	)`).Error)

	registeredID := uuid.New()
	operationalID := uuid.New()
	require.NoError(t, db.Create(&entities.User{ID: registeredID, Name: "Registered", Email: "registered@example.test", Password: "password123", Role: "user"}).Error)
	require.NoError(t, db.Create(&entities.User{ID: operationalID, Name: "Operational", Email: "operational@example.test", Password: "password123", Role: "teknik", Unit: "ULP Taman"}).Error)

	documentID := uuid.New()
	createdAt := time.Date(2026, time.September, 30, 8, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&entities.Document{
		ID: documentID, Type: "account_verification", FilePath: "documents/private", OriginalFilename: "identitas.pdf",
		MimeType: "application/pdf", SizeBytes: 42, ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Source: "uploaded", Classification: "restricted", ScanStatus: "clean", Revision: 1, UploadedBy: registeredID,
		Timestamp: entities.Timestamp{CreatedAt: createdAt, UpdatedAt: createdAt},
	}).Error)
	require.NoError(t, db.Create(&entities.AccountDocument{ID: uuid.New(), UserID: registeredID, DocumentID: documentID, DocumentType: "account_verification"}).Error)

	service := NewUserService(userRepository.NewUserRepository(db), nil, documentRepository.NewAccountDocumentRepository(db), db)
	users, total, err := service.ListAccounts(context.Background(), &query.UserFilter{BaseFilter: pagination.BaseFilter{Pagination: pagination.PaginationRequest{Page: 1, PerPage: 10}}})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	byID := make(map[string]query.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}
	require.NotNil(t, byID[registeredID.String()].AccountDocumentURL)
	require.Equal(t, "/api/documents/"+documentID.String()+"/download", *byID[registeredID.String()].AccountDocumentURL)
	require.Nil(t, byID[operationalID.String()].AccountDocumentURL)
}
