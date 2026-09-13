package controller

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

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
		Preview(ctx *gin.Context)
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
	ctx.Header("Cache-Control", "no-store")
	// Include bounded multipart overhead as well as the file itself.
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, validation.MaxUploadSizeBytes+(1<<20))
	defer func() {
		if ctx.Request.MultipartForm != nil {
			_ = ctx.Request.MultipartForm.RemoveAll()
		}
	}()
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
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrScanFailed) {
			status = http.StatusServiceUnavailable
		} else if errors.Is(err, dto.ErrMalwareDetected) {
			status = http.StatusUnprocessableEntity
		}
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_UPLOAD_DOCUMENT, err.Error(), nil)
		ctx.JSON(status, res)
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
	userID := ctx.MustGet("user_id").(string)
	docID := ctx.Param("id")

	document, err := c.documentService.Download(ctx, userID, docID)
	if err != nil {
		writeDocumentError(ctx, dto.MESSAGE_FAILED_GET_DOCUMENT, err)
		return
	}

	streamDocument(ctx, document, "attachment")
}

func (c *documentController) Preview(ctx *gin.Context) {
	userID := ctx.MustGet("user_id").(string)
	document, err := c.documentService.Preview(ctx, userID, ctx.Param("id"))
	if err != nil {
		writeDocumentError(ctx, dto.MESSAGE_FAILED_PREVIEW_DOCUMENT, err)
		return
	}
	streamDocument(ctx, document, "inline")
}

func writeDocumentError(ctx *gin.Context, message string, err error) {
	status := http.StatusBadRequest
	responseError := err.Error()
	if errors.Is(err, dto.ErrDocumentNotFound) {
		status = http.StatusNotFound
	} else if errors.Is(err, dto.ErrDocumentUnavailable) {
		status = http.StatusConflict
	} else if errors.Is(err, dto.ErrDocumentStorageUnavailable) {
		status = http.StatusServiceUnavailable
		responseError = dto.ErrDocumentStorageUnavailable.Error()
	}
	ctx.JSON(status, utils.BuildResponseFailed(message, responseError, nil))
}

func streamDocument(ctx *gin.Context, document service.DocumentContent, disposition string) {
	defer document.Body.Close()

	ctx.Header("Content-Type", document.MimeType)
	ctx.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": document.Filename}))
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")
	if document.SizeBytes >= 0 {
		ctx.Header("Content-Length", strconv.FormatInt(document.SizeBytes, 10))
	}
	ctx.Status(http.StatusOK)
	if _, err := io.Copy(ctx.Writer, document.Body); err != nil {
		_ = ctx.Error(err)
	}
}
