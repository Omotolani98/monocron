package cmd

import (
	"context"
	"net"
	"net/http"

	"github.com/Omotolani98/monocrond/handler"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const socketFile = "/var/run/monocron.sock"
const tempFile = "/tmp/monocron.sock"

type ScheduleRequest struct {
	Name     string                 `json:"name"`      // job name
	Cron     string                 `json:"cron"`      // e.g. "*/5 * * * *"
	Timezone string                 `json:"timezone"`  // optional, e.g. "Africa/Lagos"
}

type ScheduleResponse struct {
	Status  string `json:"status"`
	JobID   string `json:"job_id,omitempty"`
	Message string `json:"message,omitempty"`
}

func Start(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "start",
		Short: "starts the monocron daemon to accept sockets requests",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Hiii!!! I am a monocron daemon")
			sock, err := net.Listen("unix", tempFile)
			if err != nil {
				log.Errorf("Error on socket connection <|::|> %v", err)
				panic(err)
			}
			defer sock.Close()

			//TODO: We need to add mux server to handle path then
			mux := http.NewServeMux()
			mux.HandleFunc("/healthz", handler.Health)
			mux.HandleFunc("/schedule", handler.Schedule)
		
			srv := &http.Server{Handler: mux}
			if err := srv.Serve(sock); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
			// for {
			// 	conn, _ := sock.Accept()
			// 	go func(c net.Conn) {
			// 		defer c.Close()
			// 		buf := make([]byte, 4096)
			// 		n, _ := c.Read(buf)
			// 		log.Infof("I read ::::::: %s", string(buf[:n]))
			// 	}(conn)
			// }
		},
	}

	return cmd
}
