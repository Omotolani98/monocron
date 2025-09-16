package controller

import (
	"context"
	"strings"
	"time"

	"encore.app/controller/db"
)

type RunItem struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	TaskName    string    `json:"task_name"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Status      string    `json:"status"`
	Source      string    `json:"source"`
}

type ListRunsResponse struct {
	Runs []RunItem `json:"runs"`
}

type ListRunsParams struct {
	Status *string `query:"status"` // QUEUED|RUNNING|COMPLETED|FAILED
	Limit  *int    `query:"limit"`  // default 200, max 1000
}

func ListRuns(ctx context.Context, p *ListRunsParams) (*ListRunsResponse, error) {
	// Defaults
	limit := int32(200)
	status := ""
	if p != nil {
		if p.Limit != nil {
			v := *p.Limit
			if v > 0 && v <= 1000 {
				limit = int32(v)
			}
		}
		if p.Status != nil {
			status = strings.ToUpper(strings.TrimSpace(*p.Status))
		}
	}

	out := make([]RunItem, 0, 64)
	if status == "" {
		rows, err := q.ListRecentRuns(ctx, limit)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, RunItem{
				ID:          r.ID.String(),
				TaskID:      r.TaskID.String(),
				TaskName:    r.TaskName,
				ScheduledAt: r.ScheduledAt,
				Status:      r.Status,
				Source:      r.Source,
			})
		}
	} else {
		rows, err := q.ListRunsByStatus(ctx, db.ListRunsByStatusParams{Status: status, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, RunItem{
				ID:          r.ID.String(),
				TaskID:      r.TaskID.String(),
				TaskName:    r.TaskName,
				ScheduledAt: r.ScheduledAt,
				Status:      r.Status,
				Source:      r.Source,
			})
		}
	}
	return &ListRunsResponse{Runs: out}, nil
}
