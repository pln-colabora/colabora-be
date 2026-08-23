package main

import (
	"log"
	"os"

	"github.com/pln-colabora/colabora-be/docs"
	"github.com/pln-colabora/colabora-be/health"
	"github.com/pln-colabora/colabora-be/middlewares"
	"github.com/pln-colabora/colabora-be/modules/auth"
	"github.com/pln-colabora/colabora-be/modules/permohonan"
	"github.com/pln-colabora/colabora-be/modules/user"
	"github.com/pln-colabora/colabora-be/providers"
	"github.com/pln-colabora/colabora-be/script"
	"github.com/samber/do"

	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
)

func args(injector *do.Injector) bool {
	if len(os.Args) > 1 {
		flag := script.Commands(injector)
		return flag
	}

	return true
}

func run(server *gin.Engine) {
	server.Static("/assets", "./assets")

	port := os.Getenv("GOLANG_PORT")
	if port == "" {
		port = "8888"
	}

	var serve string
	if os.Getenv("APP_ENV") == "localhost" {
		serve = "0.0.0.0:" + port
	} else {
		serve = ":" + port
	}

	myFigure := figure.NewColorFigure("COLABORA", "", "green", true)
	myFigure.Print()

	if err := server.Run(serve); err != nil {
		log.Fatalf("error running server: %v", err)
	}
}

func main() {
	var (
		injector = do.New()
	)

	providers.RegisterDependencies(injector)

	if !args(injector) {
		return
	}

	server := gin.Default()
	server.Use(middlewares.CORSMiddleware())

	// Register module routes
	user.RegisterRoutes(server, injector)
	auth.RegisterRoutes(server, injector)
	permohonan.RegisterRoutes(server, injector)
	docs.RegisterRoutes(server)
	health.RegisterRoutes(server, injector)

	run(server)
}
