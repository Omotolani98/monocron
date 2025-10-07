package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func Stop(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "stop",
		Short: "Gracefully shuts down socket server",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Gracefully Shutting down the server")
			
			os.Remove(tempFile)
			os.Exit(1)
		},
	}

	return cmd
}
