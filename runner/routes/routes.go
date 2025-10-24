package routes

import (
	"github.com/Omotolani98/runner/handlers"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1/monocron")
	api.Get("health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  fiber.StatusOK,
			"message": "Healthy OK ✅",
		})
	})

	schedules := api.Group("/schedules")
	schedules.Post("", handlers.Schedule)
	schedules.Get("", handlers.ListSchedules)
	schedules.Post("/webhook", handlers.WebhookHandler)
}
