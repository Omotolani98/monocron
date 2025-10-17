package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Omotolani98/runner/client"
	"github.com/Omotolani98/runner/db"
	"github.com/Omotolani98/runner/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Schedule(s *models.ScheduleRequest) (*models.EntryView, error) {
	c := client.InitMuxClient()

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("Could not encode to json |> %v", err)
	}

	resp, err := c.Post("http://unix/schedule", fiber.MIMEApplicationJSON, &buf)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()

	var result models.EntryView
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.UUID = uuid.New()
	err = gorm.G[models.EntryView](db.DB).Create(context.Background(), &result)
	if err != nil {
		return nil, fmt.Errorf("Could not persist data |> %v", err)
	}

	return &result, nil
}

func ListSchedules() ([]models.EntryView, error) {
	c := client.InitMuxClient()

	resp, err := c.Get("http://unix/list")
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()

	var result []models.EntryView
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
