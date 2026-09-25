package controller

import (
	"net/http"

	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/user/dto"
	"github.com/pln-colabora/colabora-be/modules/user/query"
	"github.com/pln-colabora/colabora-be/modules/user/service"
	"github.com/pln-colabora/colabora-be/modules/user/validation"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	UserController interface {
		Me(ctx *gin.Context)
		GetAllUser(ctx *gin.Context)
		GetAllVendor(ctx *gin.Context)
		CreateAccount(ctx *gin.Context)
		UpdateAccount(ctx *gin.Context)
		Update(ctx *gin.Context)
		Delete(ctx *gin.Context)
	}

	userController struct {
		userService    service.UserService
		userValidation *validation.UserValidation
		db             *gorm.DB
	}
)

func NewUserController(injector *do.Injector, us service.UserService) UserController {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	userValidation := validation.NewUserValidation()
	return &userController{
		userService:    us,
		userValidation: userValidation,
		db:             db,
	}
}

func (c *userController) CreateAccount(ctx *gin.Context) {
	var req dto.AccountCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil))
		return
	}
	if err := c.userValidation.ValidateAccountCreateRequest(req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, utils.BuildResponseFailed("Validation failed", err.Error(), nil))
		return
	}

	result, err := c.userService.CreateAccount(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed(dto.MESSAGE_FAILED_REGISTER_USER, err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_REGISTER_USER, result))
}

func (c *userController) UpdateAccount(ctx *gin.Context) {
	var req dto.AccountUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil))
		return
	}
	if err := c.userValidation.ValidateAccountUpdateRequest(req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, utils.BuildResponseFailed("Validation failed", err.Error(), nil))
		return
	}

	result, err := c.userService.UpdateAccount(ctx.Request.Context(), req, ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed(dto.MESSAGE_FAILED_UPDATE_USER, err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_UPDATE_USER, result))
}

func (c *userController) GetAllUser(ctx *gin.Context) {
	var filter = &query.UserFilter{}
	filter.BindPagination(ctx)

	ctx.ShouldBindQuery(filter)

	users, total, err := pagination.PaginatedQueryWithIncludable[query.User](c.db, filter)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_USER, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	paginationResponse := pagination.CalculatePagination(filter.Pagination, total)
	response := pagination.NewPaginatedResponse(http.StatusOK, dto.MESSAGE_SUCCESS_GET_LIST_USER, users, paginationResponse)
	ctx.JSON(http.StatusOK, response)
}

func (c *userController) GetAllVendor(ctx *gin.Context) {
	filter := &query.VendorFilter{}
	filter.BindPagination(ctx)
	if err := ctx.ShouldBindQuery(filter); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LIST_VENDOR, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}
	if filter.Role != "" && !rbac.IsVendorRole(filter.Role) {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LIST_VENDOR, "role must be a vendor role", nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	vendors, total, err := pagination.PaginatedQueryWithIncludable[query.Vendor](c.db, filter)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LIST_VENDOR, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	paginationResponse := pagination.CalculatePagination(filter.Pagination, total)
	response := pagination.NewPaginatedResponse(http.StatusOK, dto.MESSAGE_SUCCESS_GET_LIST_VENDOR, vendors, paginationResponse)
	ctx.JSON(http.StatusOK, response)
}

func (c *userController) Me(ctx *gin.Context) {
	userId := ctx.MustGet("user_id").(string)

	result, err := c.userService.GetUserById(ctx.Request.Context(), userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_USER, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_USER, result)
	ctx.JSON(http.StatusOK, res)
}

func (c *userController) Update(ctx *gin.Context) {
	var req dto.UserUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, res)
		return
	}

	if err := c.userValidation.ValidateUserUpdateRequest(req); err != nil {
		res := utils.BuildResponseFailed("Validation failed", err.Error(), nil)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, res)
		return
	}

	userId := ctx.Param("id")
	result, err := c.userService.Update(ctx.Request.Context(), req, userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_UPDATE_USER, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_UPDATE_USER, result)
	ctx.JSON(http.StatusOK, res)
}

func (c *userController) Delete(ctx *gin.Context) {
	userId := ctx.Param("id")

	if err := c.userService.Delete(ctx.Request.Context(), userId); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_DELETE_USER, err.Error(), nil)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_DELETE_USER, nil)
	ctx.JSON(http.StatusOK, res)
}
