package service

import (
	"context"
	"testing"
	"time"

	"github.com/Caknoooo/go-pagination"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	authRepository "github.com/pln-colabora/colabora-be/modules/auth/repository"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/user/dto"
	"github.com/pln-colabora/colabora-be/modules/user/query"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/helpers"
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

func newPasswordServiceTest(t *testing.T) (UserService, *gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, email TEXT NOT NULL, telp_number TEXT,
		password TEXT NOT NULL, role TEXT NOT NULL, unit TEXT, image_url TEXT,
		is_verified BOOLEAN, created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE refresh_tokens (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, token TEXT NOT NULL,
		expires_at DATETIME, created_at DATETIME, updated_at DATETIME
	)`).Error)

	userID := uuid.New()
	require.NoError(t, db.Create(&entities.User{ID: userID, Name: "Pengguna", Email: "pengguna@example.test", Password: "old-password", Role: "teknik", Unit: "ULP Taman", IsVerified: true}).Error)
	require.NoError(t, db.Create(&entities.RefreshToken{ID: uuid.New(), UserID: userID, Token: "old-session", ExpiresAt: time.Now().Add(time.Hour)}).Error)

	service := NewUserService(userRepository.NewUserRepository(db), authRepository.NewRefreshTokenRepository(db), nil, db)
	return service, db, userID
}

func requireStoredPassword(t *testing.T, db *gorm.DB, userID uuid.UUID, password string) {
	t.Helper()
	var hashed string
	require.NoError(t, db.Raw("SELECT password FROM users WHERE id = ?", userID.String()).Scan(&hashed).Error)
	ok, _ := helpers.CheckPassword(hashed, []byte(password))
	require.True(t, ok, "stored password should match %q", password)
}

func requireRefreshTokenCount(t *testing.T, db *gorm.DB, userID uuid.UUID, want int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM refresh_tokens WHERE user_id = ?", userID.String()).Scan(&count).Error)
	require.Equal(t, want, count)
}

func TestChangePasswordRequiresCurrentPassword(t *testing.T) {
	service, db, userID := newPasswordServiceTest(t)

	err := service.ChangePassword(context.Background(), dto.ChangePasswordRequest{CurrentPassword: "wrong-password", NewPassword: "new-password"}, userID.String())
	require.ErrorIs(t, err, dto.ErrCurrentPasswordInvalid)
	requireStoredPassword(t, db, userID, "old-password")
	requireRefreshTokenCount(t, db, userID, 1)

	err = service.ChangePassword(context.Background(), dto.ChangePasswordRequest{CurrentPassword: "old-password", NewPassword: "new-password"}, userID.String())
	require.NoError(t, err)
	requireStoredPassword(t, db, userID, "new-password")
	requireRefreshTokenCount(t, db, userID, 0)
}

func TestResetPasswordSetsNewPasswordAndRevokesSessions(t *testing.T) {
	service, db, userID := newPasswordServiceTest(t)

	err := service.ResetPassword(context.Background(), dto.AdminResetPasswordRequest{NewPassword: "reset-password"}, userID.String())
	require.NoError(t, err)
	requireStoredPassword(t, db, userID, "reset-password")
	requireRefreshTokenCount(t, db, userID, 0)

	err = service.ResetPassword(context.Background(), dto.AdminResetPasswordRequest{NewPassword: "reset-password"}, uuid.NewString())
	require.ErrorIs(t, err, dto.ErrUserNotFound)
}
