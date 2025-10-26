package main

import (
	"github.com/Omotolani98/runner/db"
	"github.com/Omotolani98/runner/logger"
	"github.com/Omotolani98/runner/routes"
	"github.com/Omotolani98/runner/utils"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2"
)

func main() {
	db.InitDB()
	logger.InitLogger()

	app := fiber.New(fiber.Config{
		AppName: utils.Logo,
	})

	routes.SetupRoutes(app)

	if err := app.Listen(":5050"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
