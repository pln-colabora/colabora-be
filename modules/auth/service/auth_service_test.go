package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	authDto "github.com/pln-colabora/colabora-be/modules/auth/dto"
	"github.com/pln-colabora/colabora-be/modules/auth/repository"
	userDto "github.com/pln-colabora/colabora-be/modules/user/dto"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
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

	users := userRepo.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	return NewAuthService(users, refreshTokens, nil, db).(*authService), db
}

func TestVerifyUserSetsIsVerified(t *testing.T) {
	service, db := newAuthServiceTest(t)
	userID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id, email, is_verified) VALUES (?, ?, ?)", userID.String(), "vendor@example.test", false).Error)

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
