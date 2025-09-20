package cmd

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	conn "github.com/Omotolani98/monocron-runner/db"
	cmdutil "github.com/Omotolani98/monocron-runner/internal/cmdUtil"
	"github.com/Omotolani98/monocron-runner/pkg/gen"
	"github.com/Omotolani98/monocron-runner/server"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func Cron(ctx context.Context) *cobra.Command {
	var tz string
	var port string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "starts cron server",
		Run: func(cmd *cobra.Command, args []string) {
			conn, err := conn.NewDB(ctx)
			if err != nil {
				log.Fatalf("failed to connect to db: %v", err)
			}
			defer conn.Close()

			loc, _ := time.LoadLocation(tz)
			m := cmdutil.NewCronManager(loc, cron.DefaultLogger, *conn)
			m.Start()
			defer m.Stop(ctx)

			lis, err := net.Listen("tcp", ":"+port)
			if err != nil {
				log.Fatalf("listen: %v", err)
			}
			grpcServer := grpc.NewServer()
			gen.RegisterSchedulerServer(grpcServer, &server.SchedulerServer{Mgr: m})
			// reflection.Register(grpcServer)

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				log.Println("gRPC Scheduler listening on :" + port)
				if err := grpcServer.Serve(lis); err != nil {
					log.Fatalf("serve: %v", err)
				}
			}()

			<-sig
			log.Println("Signal caught, stopping server…")
			grpcServer.GracefulStop()
			log.Println("Shutdown complete.")
		},
	}

	cmd.Flags().StringVarP(&tz, "tz", "t", "", "input current timezone e.g 'Africa/Lagos'")
	cmd.Flags().StringVarP(&port, "port", "p", "", "port to run scheduler e.g '50000'")
	return cmd
}
