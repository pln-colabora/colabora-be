package permohonan

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/pln-colabora/colabora-be/modules/permohonan/controller"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	permohonanController := do.MustInvoke[controller.PermohonanController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	permohonanRoutes := server.Group("/api/permohonan")
	permohonanRoutes.Use(middlewares.Authenticate(jwtService))
	{
		permohonanRoutes.POST("", permohonanController.Create)
		permohonanRoutes.GET("", permohonanController.GetAll)
		permohonanRoutes.GET("/:id/activities", permohonanController.GetActivities)
		permohonanRoutes.GET("/:id/logs", permohonanController.GetLogs)
		permohonanRoutes.GET("/:id", permohonanController.GetById)
	}
}
