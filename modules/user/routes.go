package user

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/pln-colabora/colabora-be/modules/user/controller"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	userController := do.MustInvoke[controller.UserController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	userRoutes := server.Group("/api/user")
	{
		accountManager := middlewares.RequireAccountManager(injector)
		userRoutes.GET("", middlewares.Authenticate(jwtService), accountManager, userController.GetAllUser)
		userRoutes.POST("", middlewares.Authenticate(jwtService), accountManager, userController.CreateAccount)
		userRoutes.GET("/me", middlewares.Authenticate(jwtService), userController.Me)
		userRoutes.PATCH("/:id", middlewares.Authenticate(jwtService), accountManager, userController.UpdateAccount)
		userRoutes.PUT("/:id", middlewares.Authenticate(jwtService), userController.Update)
		userRoutes.DELETE("/:id", middlewares.Authenticate(jwtService), userController.Delete)
	}

	vendorRoutes := server.Group("/api/vendor")
	vendorRoutes.GET("", middlewares.Authenticate(jwtService), userController.GetAllVendor)
}
