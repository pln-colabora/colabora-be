package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"net/http"
)

type WorkflowChangesController interface {
	SubmitPriority(*gin.Context)
	EditSurvey(*gin.Context)
	ReviseConstructionDocuments(*gin.Context)
	GetRevisions(*gin.Context)
	ListPDKBSchedules(*gin.Context)
	SavePDKBSchedule(*gin.Context)
	DeletePDKBSchedule(*gin.Context)
}

func (c *permohonanController) changes() service.WorkflowChangesService {
	return c.permohonanService.(service.WorkflowChangesService)
}
func (c *permohonanController) SubmitPriority(ctx *gin.Context) {
	submitActivityRequest(ctx, func(dto.PrioritySubmitRequest) error { return nil }, c.changes().SubmitPriority)
}
func (c *permohonanController) EditSurvey(ctx *gin.Context) {
	submitActivityRequest(ctx, func(dto.SurveyRevisionRequest) error { return nil }, c.changes().EditSurvey)
}
func (c *permohonanController) ReviseConstructionDocuments(ctx *gin.Context) {
	submitActivityRequest(ctx, func(dto.DocumentRevisionRequest) error { return nil }, c.changes().ReviseConstructionDocuments)
}
func changesResult(ctx *gin.Context, result any, err error) {
	if err != nil {
		writeActivityError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Berhasil", result))
}
func (c *permohonanController) GetRevisions(ctx *gin.Context) {
	result, err := c.changes().GetRevisions(ctx, ctx.Param("id"), ctx.Param("workflow_node"), ctx.MustGet("user_id").(string))
	changesResult(ctx, result, err)
}
func (c *permohonanController) ListPDKBSchedules(ctx *gin.Context) {
	result, err := c.changes().ListPDKBSchedules(ctx, ctx.MustGet("user_id").(string), ctx.Query("from"), ctx.Query("to"))
	changesResult(ctx, result, err)
}
func (c *permohonanController) SavePDKBSchedule(ctx *gin.Context) {
	var req dto.PDKBScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeActivityError(ctx, err)
		return
	}
	result, err := c.changes().SavePDKBSchedule(ctx, ctx.Param("schedule_id"), ctx.MustGet("user_id").(string), req)
	changesResult(ctx, result, err)
}
func (c *permohonanController) DeletePDKBSchedule(ctx *gin.Context) {
	err := c.changes().DeletePDKBSchedule(ctx, ctx.Param("schedule_id"), ctx.MustGet("user_id").(string))
	changesResult(ctx, nil, err)
}
