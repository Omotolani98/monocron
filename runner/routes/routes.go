package routes

import "github.com/gofiber/fiber/v2"

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Get("health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  fiber.StatusOK,
			"message": "Healthy OK ✅",
		})
	})
}
