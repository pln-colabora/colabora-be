package controller

import (
	"errors"
	"net/http"

	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/pln-colabora/colabora-be/modules/permohonan/validation"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	PermohonanController interface {
		Create(ctx *gin.Context)
		GetAll(ctx *gin.Context)
		GetById(ctx *gin.Context)
	}

	permohonanController struct {
		permohonanService    service.PermohonanService
		permohonanValidation *validation.PermohonanValidation
		db                   *gorm.DB
	}
)

func NewPermohonanController(injector *do.Injector, s service.PermohonanService) PermohonanController {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	permohonanValidation := validation.NewPermohonanValidation()
	return &permohonanController{
		permohonanService:    s,
		permohonanValidation: permohonanValidation,
		db:                   db,
	}
}

func (c *permohonanController) Create(ctx *gin.Context) {
	var req dto.PermohonanCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	if err := c.permohonanValidation.ValidatePermohonanCreateRequest(req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	userId := ctx.MustGet("user_id").(string)

	result, err := c.permohonanService.Create(ctx, req, userId)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrOnlyPelayananPelangganCanCreate) {
			status = http.StatusForbidden
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_CREATE_PERMOHONAN, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_CREATE_PERMOHONAN, result)
	ctx.JSON(http.StatusCreated, res)
}

func (c *permohonanController) GetAll(ctx *gin.Context) {
	filter := &query.PermohonanFilter{}
	filter.BindPagination(ctx)
	ctx.ShouldBindQuery(filter)

	userId := ctx.MustGet("user_id").(string)

	results, total, err := c.permohonanService.List(ctx, filter, userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LIST_PERMOHONAN, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	paginationResponse := pagination.CalculatePagination(filter.Pagination, total)
	response := pagination.NewPaginatedResponse(http.StatusOK, dto.MESSAGE_SUCCESS_GET_LIST_PERMOHONAN, results, paginationResponse)
	ctx.JSON(http.StatusOK, response)
}

func (c *permohonanController) GetById(ctx *gin.Context) {
	id := ctx.Param("id")
	userId := ctx.MustGet("user_id").(string)

	result, err := c.permohonanService.GetById(ctx, id, userId)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrPermohonanNotFound) {
			status = http.StatusNotFound
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_PERMOHONAN, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_PERMOHONAN, result)
	ctx.JSON(http.StatusOK, res)
}
