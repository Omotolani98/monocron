package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) {
	rootCmd := &cobra.Command{
		Use:   "monocrond",
		Short: "creates a cron on the machine",
	}

	rootCmd.AddCommand(Start(ctx))
	rootCmd.AddCommand(Stop(ctx))
	if err := fang.Execute(ctx, rootCmd); err != nil {
		os.Exit(1)
	}
}
