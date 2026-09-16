package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	permohonanRepo "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	"github.com/pln-colabora/colabora-be/modules/user/dto"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/samber/do"
	"gorm.io/gorm"
)

// RequireActivityOwner is a compatibility helper for numbered endpoints. New workflow
// endpoints should authorize their canonical node directly with rbac.AuthorizeWorkflowNode.
func RequireActivityOwner(activityNumber int16, injector *do.Injector) gin.HandlerFunc {
	userRepository := do.MustInvoke[userRepo.UserRepository](injector)
	permohonanRepository := do.MustInvoke[permohonanRepo.PermohonanRepository](injector)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	return func(ctx *gin.Context) {
		userId := ctx.MustGet("user_id").(string)

		user, err := userRepository.GetUserById(ctx.Request.Context(), db, userId)
		if err != nil {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, err.Error(), nil)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response)
			return
		}

		permohonan, err := permohonanRepository.GetById(ctx.Request.Context(), db, ctx.Param("id"))
		if err != nil {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, err.Error(), nil)
			ctx.AbortWithStatusJSON(http.StatusNotFound, response)
			return
		}

		node, ok := workflow.CodeForActivity(activityNumber)
		owns := ok && rbac.OwnsWorkflowNode(user.Role, user.Unit, permohonan.JenisSambungan, permohonan.UlpUnit, node)
		if !owns {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_DENIED_ACCESS, "user does not own this activity", nil)
			ctx.AbortWithStatusJSON(http.StatusForbidden, response)
			return
		}

		ctx.Next()
	}
}

// RequireAccountManager authorizes the current database role, rather than the
// role embedded in a possibly stale access token.
func RequireAccountManager(injector *do.Injector) gin.HandlerFunc {
	userRepository := do.MustInvoke[userRepo.UserRepository](injector)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	return func(ctx *gin.Context) {
		user, err := userRepository.GetUserById(ctx.Request.Context(), db, ctx.MustGet("user_id").(string))
		if err != nil {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, err.Error(), nil)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response)
			return
		}
		if !rbac.CanManageAccounts(user.Role) {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_DENIED_ACCESS, "account management requires admin or super-user", nil)
			ctx.AbortWithStatusJSON(http.StatusForbidden, response)
			return
		}
		ctx.Next()
	}
}
