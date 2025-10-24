package handlers

import (
	"github.com/Omotolani98/runner/models"
	"github.com/Omotolani98/runner/pkg"
	"github.com/gofiber/fiber/v2"
)

func WebhookHandler(c *fiber.Ctx) error {
	gh := new(models.GhPushEvent)
	err := c.BodyParser(gh)
	if err != nil {
		return c.JSON(&fiber.Map{
			"status":  fiber.ErrBadRequest,
			"message": "Invalid Request Body",
		})
	}

	w, err := pkg.Webhook(gh)
	if err != nil {
		return c.JSON(&fiber.Map{
			"status":  fiber.ErrInternalServerError,
			"message": "could not schedule job",
		})
	}

	return c.JSON(w)
}
