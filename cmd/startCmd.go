package cmd

import (
	"context"
	"net"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const socketFile = "/var/run/monocron.sock"
const tempFile = "/tmp/monocron.sock"
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

			for {
				conn, _ := sock.Accept()
				go func(c net.Conn) {
					defer c.Close()
					buf := make([]byte, 4096)
					n, _ := c.Read(buf)
					log.Infof("I read ::::::: %s", string(buf[:n]))
				}(conn)
			}
		},
	}

	return cmd
}
