package handlers

import (
	"github.com/Omotolani98/runner/models"
	"github.com/Omotolani98/runner/pkg"
	"github.com/gofiber/fiber/v2"
)

func Schedule(c *fiber.Ctx) error {
	s := new(models.ScheduleRequest)
	err := c.BodyParser(s)
	if err != nil {
		return c.JSON(&fiber.Map{
			"status":  fiber.ErrBadRequest,
			"message": "Invalid Request Body",
		})
	}

	e, err := pkg.Schedule(s)
	if err != nil {
		return c.JSON(&fiber.Map{
			"status":  fiber.ErrInternalServerError,
			"message": "could not schedule job",
		})
	}

	return c.JSON(e)
}

func ListSchedules(c *fiber.Ctx) error {
	e, err := pkg.ListSchedules()
	if err != nil {
		return c.JSON(&fiber.Map{
			"status":  fiber.ErrInternalServerError,
			"message": "could not list schedules",
		})
	}

	return c.JSON(e)
}
