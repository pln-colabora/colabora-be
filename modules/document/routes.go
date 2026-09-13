package document

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/pln-colabora/colabora-be/modules/document/controller"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
	"gorm.io/gorm"
)

// A top-level /api/documents group owns an individual document's upload and read routes.
// The nested permohonan route is only for listing evidence attached to one request.
func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	documentController := do.MustInvoke[controller.DocumentController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	documentUploadRoutes := server.Group("/api/documents")
	documentUploadRoutes.Use(middlewares.Authenticate(jwtService))
	{
		documentUploadRoutes.POST("", documentController.Upload)
		documentUploadRoutes.GET("/:id/preview", documentController.Preview)
		documentUploadRoutes.GET("/:id/download", documentController.Download)
	}

	permohonanDocumentRoutes := server.Group("/api/permohonan/:id/documents")
	permohonanDocumentRoutes.Use(middlewares.Authenticate(jwtService))
	permohonanDocumentRoutes.Use(middlewares.RequirePermohonanAccess(do.MustInvokeNamed[*gorm.DB](injector, constants.DB)))
	{
		permohonanDocumentRoutes.GET("", documentController.List)
	}
}
