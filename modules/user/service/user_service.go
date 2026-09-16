package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	authRepository "github.com/pln-colabora/colabora-be/modules/auth/repository"
	"github.com/pln-colabora/colabora-be/modules/user/dto"
	"github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

type UserService interface {
	GetUserById(ctx context.Context, userId string) (dto.UserResponse, error)
	Update(ctx context.Context, req dto.UserUpdateRequest, userId string) (dto.UserUpdateResponse, error)
	CreateAccount(ctx context.Context, req dto.AccountCreateRequest) (dto.UserResponse, error)
	UpdateAccount(ctx context.Context, req dto.AccountUpdateRequest, userId string) (dto.UserUpdateResponse, error)
	Delete(ctx context.Context, userId string) error
}

type userService struct {
	userRepository         repository.UserRepository
	refreshTokenRepository authRepository.RefreshTokenRepository
	db                     *gorm.DB
}

func NewUserService(
	userRepo repository.UserRepository,
	refreshTokenRepo authRepository.RefreshTokenRepository,
	db *gorm.DB,
) UserService {
	return &userService{
		userRepository:         userRepo,
		refreshTokenRepository: refreshTokenRepo,
		db:                     db,
	}
}

func (s *userService) CreateAccount(ctx context.Context, req dto.AccountCreateRequest) (dto.UserResponse, error) {
	if _, exists, err := s.userRepository.CheckEmail(ctx, s.db, req.Email); err != nil && err != gorm.ErrRecordNotFound {
		return dto.UserResponse{}, err
	} else if exists {
		return dto.UserResponse{}, dto.ErrEmailAlreadyExists
	}

	user := entities.User{ID: uuid.New(), Name: req.Name, Email: req.Email, TelpNumber: req.TelpNumber, Password: req.Password, Role: req.Role, Unit: req.Unit, IsVerified: true}
	created, err := s.userRepository.Register(ctx, s.db, user)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return toUserResponse(created), nil
}

func (s *userService) UpdateAccount(ctx context.Context, req dto.AccountUpdateRequest, userID string) (dto.UserUpdateResponse, error) {
	user, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil {
		return dto.UserUpdateResponse{}, dto.ErrUserNotFound
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.TelpNumber != nil {
		user.TelpNumber = *req.TelpNumber
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Unit != nil {
		user.Unit = *req.Unit
	}
	if !rbac.ValidateRoleUnit(user.Role, user.Unit) {
		return dto.UserUpdateResponse{}, dto.ErrAccountRoleInvalid
	}

	roleOrUnitChanged := req.Role != nil || req.Unit != nil
	var updated entities.User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated, err = s.userRepository.Update(ctx, tx, user)
		if err != nil {
			return err
		}
		if roleOrUnitChanged {
			return s.refreshTokenRepository.DeleteByUserID(ctx, tx, updated.ID.String())
		}
		return nil
	})
	if err != nil {
		return dto.UserUpdateResponse{}, err
	}
	return toUserUpdateResponse(updated), nil
}

func (s *userService) GetUserById(ctx context.Context, userId string) (dto.UserResponse, error) {
	user, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.UserResponse{}, err
	}

	return toUserResponse(user), nil
}

func (s *userService) Update(ctx context.Context, req dto.UserUpdateRequest, userId string) (dto.UserUpdateResponse, error) {
	user, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.UserUpdateResponse{}, dto.ErrUserNotFound
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.TelpNumber != "" {
		user.TelpNumber = req.TelpNumber
	}

	updatedUser, err := s.userRepository.Update(ctx, s.db, user)
	if err != nil {
		return dto.UserUpdateResponse{}, err
	}

	return toUserUpdateResponse(updatedUser), nil
}

func toUserResponse(user entities.User) dto.UserResponse {
	return dto.UserResponse{ID: user.ID.String(), Name: user.Name, Email: user.Email, TelpNumber: user.TelpNumber, Role: user.Role, Unit: user.Unit, ImageUrl: user.ImageUrl, IsVerified: user.IsVerified}
}

func toUserUpdateResponse(user entities.User) dto.UserUpdateResponse {
	return dto.UserUpdateResponse{ID: user.ID.String(), Name: user.Name, TelpNumber: user.TelpNumber, Role: user.Role, Unit: user.Unit, Email: user.Email, IsVerified: user.IsVerified}
}

func (s *userService) Delete(ctx context.Context, userId string) error {
	return s.userRepository.Delete(ctx, s.db, userId)
}
