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
		permohonanRoutes.POST("/:id/survei", permohonanController.SubmitSurvey)
		permohonanRoutes.POST("/:id/rab-kko-kkf", permohonanController.SubmitRAB)
		permohonanRoutes.POST("/:id/permohonan-perluasan", permohonanController.SubmitExpansion)
		permohonanRoutes.POST("/:id/wo-vendor/tiang", permohonanController.SubmitWOTiang)
		permohonanRoutes.POST("/:id/wo-vendor/konstruksi", permohonanController.SubmitWOConstruction)
		permohonanRoutes.POST("/:id/wo-vendor/app", permohonanController.SubmitWOAPP)
		permohonanRoutes.POST("/:id/reservasi-material", permohonanController.SubmitReservationTera)
		permohonanRoutes.POST("/:id/pk-vendor", permohonanController.SubmitPKVendor)
		permohonanRoutes.POST("/:id/wo-pdkb", permohonanController.SubmitWOPDKB)
		permohonanRoutes.POST("/:id/pelaksanaan-konstruksi", permohonanController.SubmitConstructionExecution)
		permohonanRoutes.POST("/:id/pdkb-dokumentasi", permohonanController.SubmitPDKBDocumentation)
		permohonanRoutes.POST("/:id/energize-jaringan", permohonanController.SubmitEnergize)
		permohonanRoutes.POST("/:id/pemasangan-sr-app", permohonanController.SubmitSRAPP)
		permohonanRoutes.GET("", permohonanController.GetAll)
		permohonanRoutes.GET("/:id/activities", permohonanController.GetActivities)
		permohonanRoutes.GET("/:id/logs", permohonanController.GetLogs)
		permohonanRoutes.GET("/:id", permohonanController.GetById)
	}
}
