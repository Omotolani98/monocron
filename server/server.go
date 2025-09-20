package server

import (
	"context"
	"time"

	cmdutil "github.com/Omotolani98/monocron-runner/internal/cmdUtil"
	"github.com/Omotolani98/monocron-runner/pkg/gen"
	"github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SchedulerServer struct {
	gen.UnimplementedSchedulerServer
	Mgr *cmdutil.CronManager
}

func (s *SchedulerServer) AddJob(ctx context.Context, req *gen.CmdJobSpec) (*gen.AddJobResponse, error) {
	job := cmdutil.CmdJob{
		Name:    req.GetName(),
		Command: req.GetArgv(),
		Timeout: time.Duration(req.GetTimeoutSeconds()) * time.Second,
		Do: func(ctx context.Context, name string, argv []string, timeout time.Duration) error {
			err := cmdutil.RunCommand(ctx, argv)
			if err != nil {
				log.Errorf("Job FAILED [%v]", err)
				return err
			}

			log.Info("JOB SUCCESS")
			return nil
		},
	}
	res, err := s.Mgr.AddOrReplace(req.GetJobId(), req.GetSpecs(), job)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	out := &gen.AddJobResponse{JobId: res.JobID, EntryIds: toInt32(res.EntryIDs)}
	return out, nil
}

func (s *SchedulerServer) RemoveJob(ctx context.Context, req *gen.RemoveJobRequest) (*gen.RemoveJobResponse, error) {
	removed := s.Mgr.Remove(req.GetJobId())
	return &gen.RemoveJobResponse{Removed: removed}, nil
}

func (s *SchedulerServer) ListJobs(ctx context.Context, req *gen.ListJobsRequest) (*gen.ListJobsResponse, error) {
	summaries, err := s.Mgr.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list jobs: %v", err)
	}

	resp := &gen.ListJobsResponse{}
	for _, j := range summaries {
		resp.Jobs = append(resp.Jobs, &gen.Job{
			Id:          j.JobID,
			EntryId:     int32(j.EntryID),
			Name:        j.Name,
			Status:      j.Status,
			CronSpec:    j.CronSpec,
			ScheduledAt: j.ScheduledAt.Format(time.RFC3339),
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		})
	}

	return resp, nil
}

func toInt32(ids []int) []int32 {
	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}
	return out
}
