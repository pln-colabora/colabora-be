package controller

import (
	"errors"
	"net/http"

	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/pln-colabora/colabora-be/modules/permohonan/validation"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	PermohonanController interface {
		Create(ctx *gin.Context)
		GetAll(ctx *gin.Context)
		GetById(ctx *gin.Context)
		GetActivities(ctx *gin.Context)
		GetLogs(ctx *gin.Context)
		SubmitSurvey(ctx *gin.Context)
		SubmitRAB(ctx *gin.Context)
		SubmitExpansion(ctx *gin.Context)
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

func (c *permohonanController) SubmitSurvey(ctx *gin.Context) {
	var req dto.SurveySubmitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	if err := c.permohonanValidation.ValidateSurveySubmitRequest(req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	result, err := c.permohonanService.SubmitSurvey(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string), req)
	if err != nil {
		writeActivityError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_SUBMIT_ACTIVITY, result))
}

func (c *permohonanController) SubmitRAB(ctx *gin.Context) {
	var req dto.RABSubmitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	if err := c.permohonanValidation.ValidateRABSubmitRequest(req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	result, err := c.permohonanService.SubmitRAB(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string), req)
	if err != nil {
		writeActivityError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_SUBMIT_ACTIVITY, result))
}

func (c *permohonanController) SubmitExpansion(ctx *gin.Context) {
	var req dto.ExpansionSubmitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	if err := c.permohonanValidation.ValidateExpansionSubmitRequest(req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	result, err := c.permohonanService.SubmitExpansion(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string), req)
	if err != nil {
		writeActivityError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_SUBMIT_ACTIVITY, result))
}

func writeActivityError(ctx *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, rbac.ErrWorkflowForbidden):
		status = http.StatusForbidden
	case errors.Is(err, dto.ErrPermohonanNotFound), errors.Is(err, documentDTO.ErrDocumentNotFound):
		status = http.StatusNotFound
	case errors.Is(err, workflow.ErrNotActionable), errors.Is(err, workflow.ErrInvalidState),
		errors.Is(err, documentDTO.ErrDocumentAlreadyAttached):
		status = http.StatusConflict
	case errors.Is(err, dto.ErrSubmitActivity):
		status = http.StatusInternalServerError
	}
	ctx.JSON(status, utils.BuildResponseFailed(dto.MESSAGE_FAILED_SUBMIT_ACTIVITY, err.Error(), nil))
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
		if errors.Is(err, dto.ErrCreateForbidden) {
			status = http.StatusForbidden
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_CREATE_PERMOHONAN, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_CREATE_PERMOHONAN, result)
	ctx.JSON(http.StatusCreated, res)
}

func (c *permohonanController) GetActivities(ctx *gin.Context) {
	results, err := c.permohonanService.GetActivities(ctx, ctx.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrPermohonanNotFound) {
			status = http.StatusNotFound
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_ACTIVITIES, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}
	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_ACTIVITIES, results)
	ctx.JSON(http.StatusOK, res)
}

func (c *permohonanController) GetLogs(ctx *gin.Context) {
	results, err := c.permohonanService.GetLogs(ctx, ctx.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrPermohonanNotFound) {
			status = http.StatusNotFound
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LOGS, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}
	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_LOGS, results)
	ctx.JSON(http.StatusOK, res)
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
