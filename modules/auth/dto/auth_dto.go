package dto

import (
	"errors"
)

const (
	MESSAGE_FAILED_REFRESH_TOKEN        = "failed refresh token"
	MESSAGE_SUCCESS_REFRESH_TOKEN       = "success refresh token"
	MESSAGE_FAILED_LOGOUT               = "failed logout"
	MESSAGE_SUCCESS_LOGOUT              = "success logout"
	MESSAGE_FAILED_SEND_PASSWORD_RESET  = "failed send password reset"
	MESSAGE_SUCCESS_SEND_PASSWORD_RESET = "success send password reset"
	MESSAGE_FAILED_RESET_PASSWORD       = "failed reset password"
	MESSAGE_SUCCESS_RESET_PASSWORD      = "success reset password"
	MESSAGE_FAILED_VERIFY_USER          = "failed verify user"
	MESSAGE_SUCCESS_VERIFY_USER         = "success verify user"
	MESSAGE_SUCCESS_GET_ROLES           = "success get roles"
)

var (
	ErrRefreshTokenNotFound            = errors.New("refresh token not found")
	ErrRefreshTokenExpired             = errors.New("refresh token expired")
	ErrInvalidCredentials              = errors.New("invalid credentials")
	ErrAccountNotVerified              = errors.New("account is not verified")
	ErrVerificationDocument            = errors.New("verification document is required")
	ErrVerificationDocumentUnavailable = errors.New("verification document is unavailable")
	ErrPasswordResetToken              = errors.New("password reset token invalid")
	ErrInvalidUserID                   = errors.New("invalid user id")
)

type (
	RefreshTokenRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	TokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Role         string `json:"role"`
	}

	SendPasswordResetRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	ResetPasswordRequest struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	VerifyUserRequest struct {
		Role string `json:"role" binding:"required,oneof=admin user pelayanan-pelanggan teknik perencanaan konstruksi transaksi-energi jaringan nps pdkb vendor-tiang vendor-konstruksi vendor-sr-app super-user"`
	}

	VerifyUserResponse struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		Role       string `json:"role"`
		IsVerified bool   `json:"is_verified"`
	}
)
