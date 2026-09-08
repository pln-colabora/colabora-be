package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/service"
	"github.com/pln-colabora/colabora-be/modules/document/validation"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	DocumentController interface {
		Upload(ctx *gin.Context)
		List(ctx *gin.Context)
		Download(ctx *gin.Context)
	}

	documentController struct {
		documentService    service.DocumentService
		documentValidation *validation.DocumentValidation
		db                 *gorm.DB
	}
)

func NewDocumentController(injector *do.Injector, s service.DocumentService) DocumentController {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	documentValidation := validation.NewDocumentValidation()
	return &documentController{
		documentService:    s,
		documentValidation: documentValidation,
		db:                 db,
	}
}

// Upload is registered at the top-level POST /api/documents (no permohonan in the path) —
// a document is uploaded standalone and attached to a permohonan+activity later, when a
// Phase 4 activity endpoint calls DocumentService.AttachToWorkflowNode.
func (c *documentController) Upload(ctx *gin.Context) {
	var req dto.DocumentUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	if err := c.documentValidation.ValidateDocumentUploadRequest(req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	userId := ctx.MustGet("user_id").(string)

	result, err := c.documentService.Upload(ctx, userId, req)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_UPLOAD_DOCUMENT, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_UPLOAD_DOCUMENT, result)
	ctx.JSON(http.StatusCreated, res)
}

func (c *documentController) List(ctx *gin.Context) {
	permohonanId := ctx.Param("id")

	var workflowNode *string
	if value := ctx.Query("workflow_node"); value != "" {
		workflowNode = &value
	}

	results, err := c.documentService.List(ctx, permohonanId, workflowNode)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_LIST_DOCUMENT, err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_LIST_DOCUMENT, results)
	ctx.JSON(http.StatusOK, res)
}

func (c *documentController) Download(ctx *gin.Context) {
	permohonanId := ctx.Param("id")
	docId := ctx.Param("doc_id")

	url, err := c.documentService.Download(ctx, permohonanId, docId)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrDocumentNotFound) {
			status = http.StatusNotFound
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DOCUMENT, err.Error(), nil)
		ctx.JSON(status, res)
		return
	}

	ctx.Redirect(http.StatusFound, url)
}
