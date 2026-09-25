package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth/controller"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	authController := do.MustInvoke[controller.AuthController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	authRoutes := server.Group("/api/auth")
	{
		accountManager := middlewares.RequireAccountManager(injector)
		authRoutes.POST("/register", authController.Register)
		authRoutes.POST("/login", authController.Login)
		authRoutes.POST("/refresh", authController.RefreshToken)
		authRoutes.POST("/logout", authController.Logout)
		authRoutes.POST("/send-verification-email", authController.SendVerificationEmail)
		authRoutes.POST("/verify-email", authController.VerifyEmail)
		authRoutes.POST("/verify/:user_id", middlewares.Authenticate(jwtService), accountManager, authController.VerifyUser)
		authRoutes.POST("/send-password-reset", authController.SendPasswordReset)
		authRoutes.POST("/reset-password", authController.ResetPassword)
	}
}
