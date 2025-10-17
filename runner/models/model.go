package models

import (
	"time"

	"github.com/google/uuid"
)

type ScheduleRequest struct {
	Name     string        `json:"name"`
	Schedule string        `json:"schedule"`
	Timezone string        `json:"timezone"`
	Timeout  time.Duration `json:"timeout"`
	Argv     []string      `json:"argv"`
}

type EntryView struct {
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"uuid"`
	EntryID     int       `json:"id"`
	Name        string    `json:"name"`
	Schedule    string    `json:"schedule"`
	Next        time.Time `json:"next"`
	Prev        time.Time `json:"prev"`
	Description string    `json:"description"`
}
