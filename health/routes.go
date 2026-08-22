package health

import (
	"context"
	"net/http"
	"time"

	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type checkResponse struct {
	Database string `json:"database"`
}

// RegisterRoutes serves GET /health, which pings the database so the check
// reflects whether the app can actually serve requests, not just that the
// process is running.
func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	server.GET("/health", func(ctx *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			res := utils.BuildResponseFailed("service unhealthy", err.Error(), checkResponse{Database: "down"})
			ctx.JSON(http.StatusServiceUnavailable, res)
			return
		}

		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			res := utils.BuildResponseFailed("service unhealthy", err.Error(), checkResponse{Database: "down"})
			ctx.JSON(http.StatusServiceUnavailable, res)
			return
		}

		res := utils.BuildResponseSuccess("service healthy", checkResponse{Database: "up"})
		ctx.JSON(http.StatusOK, res)
	})
}
