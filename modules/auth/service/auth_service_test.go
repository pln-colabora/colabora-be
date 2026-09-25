package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	authDto "github.com/pln-colabora/colabora-be/modules/auth/dto"
	"github.com/pln-colabora/colabora-be/modules/auth/repository"
	documentRepo "github.com/pln-colabora/colabora-be/modules/document/repository"
	userDto "github.com/pln-colabora/colabora-be/modules/user/dto"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/helpers"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuthServiceTest(t *testing.T) (*authService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		name TEXT,
		email TEXT,
		telp_number TEXT,
		password TEXT,
		role TEXT,
		unit TEXT,
		image_url TEXT,
		is_verified BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE documents (
		id TEXT PRIMARY KEY,
		type TEXT,
		file_path TEXT,
		original_filename TEXT,
		mime_type TEXT,
		size_bytes INTEGER,
		checksum_sha256 TEXT,
		source TEXT,
		classification TEXT,
		scan_status TEXT,
		scan_checked_at DATETIME,
		revision INTEGER,
		supersedes_id TEXT,
		superseded_by_id TEXT,
		uploaded_by TEXT,
		permohonan_id TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE account_documents (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		document_id TEXT NOT NULL UNIQUE,
		document_type TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error)

	users := userRepo.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	accountDocuments := documentRepo.NewAccountDocumentRepository(db)
	return NewAuthService(users, refreshTokens, nil, nil, accountDocuments, db).(*authService), db
}

func addVerificationDocument(t *testing.T, db *gorm.DB, userID uuid.UUID) {
	t.Helper()
	documentID := uuid.New()
	require.NoError(t, db.Create(&entities.Document{ID: documentID, Type: "account_verification", FilePath: "documents/test", OriginalFilename: "test.pdf", MimeType: "application/pdf", SizeBytes: 4, ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Source: "uploaded", Classification: "restricted", ScanStatus: "clean", Revision: 1, UploadedBy: userID}).Error)
	require.NoError(t, db.Create(&entities.AccountDocument{ID: uuid.New(), UserID: userID, DocumentID: documentID, DocumentType: "account_verification"}).Error)
}

func TestVerifyUserSetsIsVerified(t *testing.T) {
	service, db := newAuthServiceTest(t)
	userID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id, email, is_verified) VALUES (?, ?, ?)", userID.String(), "vendor@example.test", false).Error)
	addVerificationDocument(t, db, userID)

	result, err := service.VerifyUser(context.Background(), userID.String())
	require.NoError(t, err)
	require.Equal(t, userID.String(), result.ID)
	require.Equal(t, "vendor@example.test", result.Email)
	require.True(t, result.IsVerified)

	var verified bool
	require.NoError(t, db.Raw("SELECT is_verified FROM users WHERE id = ?", userID.String()).Scan(&verified).Error)
	require.True(t, verified)
}

func TestVerifyUserIsIdempotent(t *testing.T) {
	service, db := newAuthServiceTest(t)
	userID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id, email, is_verified) VALUES (?, ?, ?)", userID.String(), "vendor@example.test", true).Error)
	addVerificationDocument(t, db, userID)

	result, err := service.VerifyUser(context.Background(), userID.String())
	require.NoError(t, err)
	require.True(t, result.IsVerified)
}

func TestVerifyUserRejectsInvalidOrMissingUser(t *testing.T) {
	service, _ := newAuthServiceTest(t)

	_, err := service.VerifyUser(context.Background(), "not-a-uuid")
	require.ErrorIs(t, err, authDto.ErrInvalidUserID)

	_, err = service.VerifyUser(context.Background(), uuid.NewString())
	require.ErrorIs(t, err, userDto.ErrUserNotFound)
}

func TestLoginRejectsUnverifiedAccount(t *testing.T) {
	service, db := newAuthServiceTest(t)
	password, err := helpers.HashPassword("password123")
	require.NoError(t, err)
	userID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id, email, password, is_verified) VALUES (?, ?, ?, ?)", userID.String(), "pending@example.test", password, false).Error)

	_, err = service.Login(context.Background(), userDto.UserLoginRequest{Email: "pending@example.test", Password: "password123"})
	require.ErrorIs(t, err, authDto.ErrAccountNotVerified)
}

func TestVerifyUserRequiresAccountDocument(t *testing.T) {
	service, db := newAuthServiceTest(t)
	userID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id, email, is_verified) VALUES (?, ?, ?)", userID.String(), "without-document@example.test", false).Error)

	_, err := service.VerifyUser(context.Background(), userID.String())
	require.ErrorIs(t, err, authDto.ErrVerificationDocument)
}
