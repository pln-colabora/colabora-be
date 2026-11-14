package controller

import (
	"errors"
	"net/http"
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
	if err != nil || file == nil || !strings.HasSuffix(strings.ToLower(file.Filename), ".xls") || file.Size > 10<<20 {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid EIS file", "file must be an XLS up to 10 MB", nil))
		return
	}
	opened, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("invalid EIS file", err.Error(), nil))
		return
	}
	defer opened.Close()
	result, err := c.eisService().ImportEIS(ctx, ctx.MustGet("user_id").(string), opened)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dto.ErrCreateForbidden) {
			status = http.StatusForbidden
		}
		ctx.JSON(status, utils.BuildResponseFailed("failed import EIS", err.Error(), nil))
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("EIS drafts imported", result))
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
