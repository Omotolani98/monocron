package models

import (
	"time"
)

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


type NamedJob interface {
	Run()
	Name() string
}

type EntryView struct {
	ID          int `json:"id"`
	Name string `json:"name"`
	Schedule    string       `json:"schedule"`
	Next        time.Time    `json:"next"`
	Prev        time.Time    `json:"prev"`
	Description string       `json:"description"`
}
