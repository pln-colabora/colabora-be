package document

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/pln-colabora/colabora-be/modules/document/controller"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
)

// Two route groups: a top-level /api/documents for standalone upload (no permohonan known
// yet), and /api/permohonan/:id/documents for reading documents once they've been attached.
// The nested group reuses the :id param name permohonan/routes.go already registers at that
// path — Gin's router requires the same wildcard name at a shared tree position.
func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	documentController := do.MustInvoke[controller.DocumentController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	documentUploadRoutes := server.Group("/api/documents")
	documentUploadRoutes.Use(middlewares.Authenticate(jwtService))
	{
		documentUploadRoutes.POST("", documentController.Upload)
	}

	permohonanDocumentRoutes := server.Group("/api/permohonan/:id/documents")
	permohonanDocumentRoutes.Use(middlewares.Authenticate(jwtService))
	{
		permohonanDocumentRoutes.GET("", documentController.List)
		permohonanDocumentRoutes.GET("/:doc_id", documentController.Download)
	}
}
