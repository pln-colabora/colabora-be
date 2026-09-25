package dto

import (
	"errors"
	"mime/multipart"
)

const (
	// Failed
	MESSAGE_FAILED_GET_DATA_FROM_BODY = "failed get data from body"
	MESSAGE_FAILED_REGISTER_USER      = "failed create user"
	MESSAGE_FAILED_GET_LIST_USER      = "failed get list user"
	MESSAGE_FAILED_GET_LIST_VENDOR    = "failed get list vendor"
	MESSAGE_FAILED_TOKEN_NOT_VALID    = "token not valid"
	MESSAGE_FAILED_TOKEN_NOT_FOUND    = "token not found"
	MESSAGE_FAILED_GET_USER           = "failed get user"
	MESSAGE_FAILED_LOGIN              = "failed login"
	MESSAGE_FAILED_UPDATE_USER        = "failed update user"
	MESSAGE_FAILED_DELETE_USER        = "failed delete user"
	MESSAGE_FAILED_PROSES_REQUEST     = "failed proses request"
	MESSAGE_FAILED_DENIED_ACCESS      = "denied access"
	MESSAGE_FAILED_VERIFY_EMAIL       = "failed verify email"

	// Success
	MESSAGE_SUCCESS_REGISTER_USER           = "success create user"
	MESSAGE_SUCCESS_GET_LIST_USER           = "success get list user"
	MESSAGE_SUCCESS_GET_LIST_VENDOR         = "success get list vendor"
	MESSAGE_SUCCESS_GET_USER                = "success get user"
	MESSAGE_SUCCESS_LOGIN                   = "success login"
	MESSAGE_SUCCESS_UPDATE_USER             = "success update user"
	MESSAGE_SUCCESS_DELETE_USER             = "success delete user"
	MESSAGE_SEND_VERIFICATION_EMAIL_SUCCESS = "success send verification email"
	MESSAGE_SUCCESS_VERIFY_EMAIL            = "success verify email"
)

var (
	ErrCreateUser             = errors.New("failed to create user")
	ErrGetUserById            = errors.New("failed to get user by id")
	ErrGetUserByEmail         = errors.New("failed to get user by email")
	ErrEmailAlreadyExists     = errors.New("email already exist")
	ErrUpdateUser             = errors.New("failed to update user")
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailNotFound          = errors.New("email not found")
	ErrDeleteUser             = errors.New("failed to delete user")
	ErrTokenInvalid           = errors.New("token invalid")
	ErrTokenExpired           = errors.New("token expired")
	ErrAccountAlreadyVerified = errors.New("account already verified")
	ErrAccountRoleInvalid     = errors.New("invalid account role or unit")
)

type (
	UserCreateRequest struct {
		Name       string                `json:"name" form:"name" binding:"required,min=2,max=100"`
		TelpNumber string                `json:"telp_number" form:"telp_number" binding:"omitempty,min=8,max=20"`
		Email      string                `json:"email" form:"email" binding:"required,email"`
		Password   string                `json:"password" form:"password" binding:"required,min=8"`
		Document   *multipart.FileHeader `json:"document" form:"document" binding:"required"`
	}

	UserResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		TelpNumber string `json:"telp_number"`
		Role       string `json:"role"`
		Unit       string `json:"unit"`
		ImageUrl   string `json:"image_url"`
		IsVerified bool   `json:"is_verified"`
	}
	UserUpdateRequest struct {
		Name       string `json:"name" form:"name" binding:"omitempty,min=2,max=100"`
		TelpNumber string `json:"telp_number" form:"telp_number" binding:"omitempty,min=8,max=20"`
		Email      string `json:"email" form:"email" binding:"omitempty,email"`
	}

	AccountCreateRequest struct {
		Name       string `json:"name" binding:"required,min=2,max=100"`
		TelpNumber string `json:"telp_number" binding:"omitempty,min=8,max=20"`
		Email      string `json:"email" binding:"required,email"`
		Password   string `json:"password" binding:"required,min=8"`
		Role       string `json:"role" binding:"required"`
		Unit       string `json:"unit"`
	}

	AccountUpdateRequest struct {
		Name       *string `json:"name" binding:"omitempty,min=2,max=100"`
		TelpNumber *string `json:"telp_number" binding:"omitempty,min=8,max=20"`
		Email      *string `json:"email" binding:"omitempty,email"`
		Role       *string `json:"role"`
		Unit       *string `json:"unit"`
	}

	UserUpdateResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		TelpNumber string `json:"telp_number"`
		Role       string `json:"role"`
		Unit       string `json:"unit"`
		Email      string `json:"email"`
		IsVerified bool   `json:"is_verified"`
	}

	SendVerificationEmailRequest struct {
		Email string `json:"email" form:"email" binding:"required"`
	}

	VerifyEmailRequest struct {
		Token string `json:"token" form:"token" binding:"required"`
	}

	VerifyEmailResponse struct {
		Email      string `json:"email"`
		IsVerified bool   `json:"is_verified"`
	}

	UserLoginRequest struct {
		Email    string `json:"email" form:"email" binding:"required"`
		Password string `json:"password" form:"password" binding:"required"`
	}
)
