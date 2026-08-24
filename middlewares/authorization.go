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
	"github.com/samber/do"
	"gorm.io/gorm"
)

// RequireActivityOwner rejects a request unless the caller's role currently owns the given
// activity number on the permohonan identified by the route's :id param. Must run after
// Authenticate (needs user_id in context) and on a route with an :id param.
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

		owns := rbac.OwnsActivity(user.Role, user.Unit, activityNumber, permohonan.JenisSambungan, permohonan.UlpUnit, permohonan.OwnerFnOverride)
		if !owns {
			response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_DENIED_ACCESS, "user does not own this activity", nil)
			ctx.AbortWithStatusJSON(http.StatusForbidden, response)
			return
		}

		ctx.Next()
	}
}
