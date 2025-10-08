package models

import "time"

type ScheduleRequest struct {
	Name string `json:"name"`
	Schedule string `json:"schedule"`
	Timezone string `json:"timezone"`
	Timeout time.Duration `json:"timeout"`
	Argv []string `json:"argv"`
}

type ScheduleResponse struct {
	EntryJobId int `json:"jobId"`
	Name string `json:"name"`
}
