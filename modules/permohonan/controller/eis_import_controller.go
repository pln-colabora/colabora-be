package controller

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/pln-colabora/colabora-be/pkg/utils"
)

func (c *permohonanController) eisService() service.EISDraftService {
	return c.permohonanService.(service.EISDraftService)
}

func (c *permohonanController) ImportEIS(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	format := ""
	if file != nil {
		format = strings.ToLower(filepath.Ext(file.Filename))
	}
	if err != nil || file == nil || (format != ".xls" && format != ".xlsx") || file.Size > 30<<20 {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid EIS file", "file must be an XLS or XLSX up to 30 MB", nil))
		return
	}
	opened, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid EIS file", err.Error(), nil))
		return
	}
	defer opened.Close()
	result, err := c.eisService().ImportEIS(ctx, ctx.MustGet("user_id").(string), format, opened)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrCreateForbidden) {
			status = http.StatusForbidden
		}
		ctx.JSON(status, utils.BuildResponseFailed("failed import EIS", err.Error(), nil))
		return
	}
	// data keeps the created drafts as before; meta reports rows skipped
	// because the same EIS application was already imported.
	response := utils.BuildResponseSuccess("EIS drafts imported", result.Created)
	response.Meta = map[string]any{
		"created_count": len(result.Created),
		"skipped_count": len(result.Skipped),
		"skipped":       result.Skipped,
	}
	ctx.JSON(http.StatusCreated, response)
}

func (c *permohonanController) EditEISDraft(ctx *gin.Context) {
	var req dto.EISDraftEditRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid draft", err.Error(), nil))
		return
	}
	result, err := c.eisService().EditEISDraft(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("failed edit draft", err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("draft saved", result))
}

func (c *permohonanController) DeleteEISDraft(ctx *gin.Context) {
	err := c.eisService().DeleteEISDraft(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string))
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrEISDraftNotFound):
			status = http.StatusNotFound
		case errors.Is(err, service.ErrEISDraftConflict):
			status = http.StatusConflict
		}
		ctx.JSON(status, utils.BuildResponseFailed("failed delete draft", err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("draft deleted", nil))
}

func (c *permohonanController) ContinueEISDraft(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid evidence", err.Error(), nil))
		return
	}
	result, err := c.eisService().ContinueEISDraft(ctx, ctx.Param("id"), ctx.MustGet("user_id").(string), form.File["evidence_files"])
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrEISDraft) {
			status = http.StatusConflict
		}
		if errors.Is(err, dto.ErrCreateForbidden) {
			status = http.StatusForbidden
		}
		ctx.JSON(status, utils.BuildResponseFailed("failed continue draft", err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("draft continued", result))
}
