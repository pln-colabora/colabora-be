package providers

import (
	"github.com/pln-colabora/colabora-be/config"
	authController "github.com/pln-colabora/colabora-be/modules/auth/controller"
	authRepo "github.com/pln-colabora/colabora-be/modules/auth/repository"
	authService "github.com/pln-colabora/colabora-be/modules/auth/service"
	permohonanController "github.com/pln-colabora/colabora-be/modules/permohonan/controller"
	permohonanRepo "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	permohonanService "github.com/pln-colabora/colabora-be/modules/permohonan/service"
	userController "github.com/pln-colabora/colabora-be/modules/user/controller"
	"github.com/pln-colabora/colabora-be/modules/user/repository"
	userService "github.com/pln-colabora/colabora-be/modules/user/service"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func InitDatabase(injector *do.Injector) {
	do.ProvideNamed(injector, constants.DB, func(i *do.Injector) (*gorm.DB, error) {
		return config.SetUpDatabaseConnection(), nil
	})
}

func RegisterDependencies(injector *do.Injector) {
	InitDatabase(injector)

	do.ProvideNamed(injector, constants.JWTService, func(i *do.Injector) (authService.JWTService, error) {
		return authService.NewJWTService(), nil
	})

	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	jwtService := do.MustInvokeNamed[authService.JWTService](injector, constants.JWTService)

	userRepository := repository.NewUserRepository(db)
	refreshTokenRepository := authRepo.NewRefreshTokenRepository(db)
	permohonanRepository := permohonanRepo.NewPermohonanRepository(db)
	slaRuleRepository := permohonanRepo.NewSLARuleRepository(db)

	userService := userService.NewUserService(userRepository, db)
	authService := authService.NewAuthService(userRepository, refreshTokenRepository, jwtService, db)
	permohonanService := permohonanService.NewPermohonanService(permohonanRepository, slaRuleRepository, userRepository, db)

	do.Provide(
		injector, func(i *do.Injector) (userController.UserController, error) {
			return userController.NewUserController(i, userService), nil
		},
	)

	do.Provide(
		injector, func(i *do.Injector) (authController.AuthController, error) {
			return authController.NewAuthController(i, authService), nil
		},
	)

	do.Provide(
		injector, func(i *do.Injector) (permohonanController.PermohonanController, error) {
			return permohonanController.NewPermohonanController(i, permohonanService), nil
		},
	)
}
