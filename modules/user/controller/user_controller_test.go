package controller

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/user/dto"
	"github.com/pln-colabora/colabora-be/modules/user/validation"
	"github.com/stretchr/testify/require"
)

type deleteUserService struct {
	deletedUserID string
	updatedUserID string
}

func (s *deleteUserService) GetUserById(context.Context, string) (dto.UserResponse, error) {
	return dto.UserResponse{}, nil
}

func (s *deleteUserService) Update(_ context.Context, _ dto.UserUpdateRequest, userID string) (dto.UserUpdateResponse, error) {
	s.updatedUserID = userID
	return dto.UserUpdateResponse{}, nil
}

func (s *deleteUserService) CreateAccount(context.Context, dto.AccountCreateRequest) (dto.UserResponse, error) {
	return dto.UserResponse{}, nil
}

func (s *deleteUserService) UpdateAccount(_ context.Context, _ dto.AccountUpdateRequest, userID string) (dto.UserUpdateResponse, error) {
	s.updatedUserID = userID
	return dto.UserUpdateResponse{}, nil
}

func (s *deleteUserService) Delete(_ context.Context, userID string) error {
	s.deletedUserID = userID
	return nil
}

func TestDeleteUsesPathUserIDRatherThanAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &deleteUserService{}
	controller := &userController{userService: service}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/user/target-user-id", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "target-user-id"}}
	ctx.Set("user_id", "authenticated-user-id")

	controller.Delete(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "target-user-id", service.deletedUserID)
}

func TestUpdateUsesPathUserIDRatherThanAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &deleteUserService{}
	controller := &userController{userService: service, userValidation: validation.NewUserValidation()}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/user/target-user-id", bytes.NewBufferString(`{"name":"Target User"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "target-user-id"}}
	ctx.Set("user_id", "authenticated-user-id")

	controller.Update(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "target-user-id", service.updatedUserID)
}

func TestGetAllVendorRejectsNonVendorRoleFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/vendor?role=admin", nil)

	controller := &userController{}
	controller.GetAllVendor(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
