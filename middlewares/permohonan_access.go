package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"gorm.io/gorm"
)

// RequirePermohonanAccess protects every HTTP resource below a request, including
// activity submissions (which also perform exact-node authorization in a transaction).
func RequirePermohonanAccess(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		if id == "" {
			ctx.Next()
			return
		} // create/list have no request id
		reject := func(status int, message string) {
			ctx.AbortWithStatusJSON(status, utils.BuildResponseFailed("request access denied", message, nil))
		}
		if _, err := uuid.Parse(id); err != nil {
			reject(http.StatusNotFound, "permohonan not found")
			return
		}
		userID, ok := ctx.Get("user_id")
		actorID, valid := userID.(string)
		if !ok || !valid {
			reject(http.StatusUnauthorized, "authentication required")
			return
		}
		var actor entities.User
		if err := db.WithContext(ctx).Where("id = ?", actorID).Take(&actor).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				reject(http.StatusUnauthorized, "authentication required")
			} else {
				reject(http.StatusInternalServerError, "unable to authorize request")
			}
			return
		}
		var count int64
		err := rbac.ApplyReadScope(db.WithContext(ctx).Model(&entities.Permohonan{}), actor.Role, actor.Unit, actor.ID.String()).
			Where("permohonan.id = ?", id).Count(&count).Error
		if err != nil {
			reject(http.StatusInternalServerError, "unable to authorize request")
			return
		}
		if count == 0 {
			reject(http.StatusNotFound, "permohonan not found")
			return
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.Next()
	}
}
