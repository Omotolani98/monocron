package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type TaskItem struct {
	Author     string    `json:"author"`
	Name       string    `json:"task"`
	Schedule   string    `json:"schedule"`
	NextAt     time.Time `json:"schedule_at"`
	LastStatus string    `json:"status"`
}

type ListTasksResponse struct {
	Tasks []TaskItem `json:"tasks"`
}

//encore:api public method=GET path=/tasks
func ListTasks(ctx context.Context) (*ListTasksResponse, error) {
	tasks, err := q.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TaskItem, 0, len(tasks))
	now := time.Now()
	for _, t := range tasks {
		// Compute next schedule time
		loc := time.UTC
		if t.Timezone != "" {
			if l, err := time.LoadLocation(t.Timezone); err == nil {
				loc = l
			}
		}
		sch, err := cron.ParseStandard(t.Schedule)
		if err != nil {
			sch = cron.Every(0) // invalid schedule; keep zero NextAt
		}
		next := sch.Next(now.In(loc)).UTC()
		// Fetch last run status
		var lastStatus string
		if err := pgxdb.QueryRow(ctx, `
            SELECT status FROM runs WHERE task_id = $1
            ORDER BY scheduled_at DESC LIMIT 1
        `, t.ID).Scan(&lastStatus); err != nil {
			lastStatus = "N/A"
		}
		out = append(out, TaskItem{
			Author:     "", // unknown at this stage
			Name:       t.Name,
			Schedule:   t.Schedule,
			NextAt:     next,
			LastStatus: normalizeStatus(lastStatus),
		})
	}
	return &ListTasksResponse{Tasks: out}, nil
}

func normalizeStatus(s string) string {
	switch s {
	case "QUEUED", "RUNNING", "COMPLETED", "FAILED":
		return s
	case "pending":
		return "QUEUED"
	case "success":
		return "COMPLETED"
	default:
		return s
	}
}

//encore:api public method=DELETE path=/tasks/:name
func DeleteTaskAPI(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name required")
	}
	return q.DeleteTask(ctx, name)
}
