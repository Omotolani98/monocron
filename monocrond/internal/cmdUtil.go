package cmdUtil

import (
	"context"
	"time"

	"github.com/Omotolani98/monocrond/config"
	"github.com/Omotolani98/monocrond/models"
	"github.com/charmbracelet/log"
)

func ScheduleCron(s models.ScheduleRequest) (*models.ScheduleResponse, error) {
	spec := s.Schedule
	if s.Timezone != "" {
		spec = "CRON_TZ=" + s.Timezone + " " + spec
	}

	to := time.Duration(s.Timeout) * time.Second
	if to <= 0 {
		to = 30 * time.Second 
	}

	id , _ := config.Cron.AddFunc(spec, func() {
		log.Info("I am a running job")

        ctx, cancel := context.WithTimeout(context.Background(), to)
		defer cancel()

		err := 	RunCommand(ctx, s.Argv)
		if err != nil {
			// set status to fail 
			log.Errorf("Error in the background <|::|> %v\n", err)
		} else {
			// set status to success
			log.Info("Job Successful <|::|>\n")
		}

	})

	config.Mu.Lock()
	config.Jobs[id] = &config.JobMeta{
		Name:      s.Name,
		Spec:      spec,
        CreatedAt: time.Now(),
        Timeout:   to,
    }
    config.Mu.Unlock()

    return &models.ScheduleResponse{
        EntryJobId: int(id),
        Name:       s.Name,
    }, nil
}
