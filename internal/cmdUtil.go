package cmdUtil

import (
	"context"
	"time"

	"github.com/Omotolani98/monocrond/config"
	"github.com/Omotolani98/monocrond/models"
	"github.com/charmbracelet/log"
	"github.com/robfig/cron/v3"
)

func ScheduleCron(s models.ScheduleRequest) (*models.ScheduleResponse, error) {
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		log.Errorf("Invalid Timezone, It's new to me <|:::|> %v\n", err)
		return nil, err
	}

	config.Cron = cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
			),
		)
	config.Cron.Start()

	id , _ := config.Cron.AddFunc(s.Schedule, func() {
		log.Info("I am a running job")

		err := 	RunCommand(context.Background(), s.Argv)
		if err != nil {
			// set status to fail 
			log.Errorf("Error in the background <|::|> %v\n", err)
		} else {
			// set status to success
			log.Info("Job Successful <|::|>\n")
		}

	})
	config.Cron.Start()

	return &models.ScheduleResponse{
		EntryJobId: int(id),
		Name: s.Name,
	}, nil
}
