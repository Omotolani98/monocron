package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func Ctl(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts server so ",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	return cmd
}
