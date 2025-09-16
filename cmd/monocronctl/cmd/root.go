package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	apiBase string
)

var rootCmd = &cobra.Command{
	Use:   "monocronctl",
	Short: "Monocron controller CLI",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiBase, "api", defaultAPI(), "Base URL to the controller API")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(runnersCmd)
}

func defaultAPI() string {
	if v := os.Getenv("MONOCRON_API"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:4000/controller"
}

func Execute(ctx context.Context) {
	// if err := rootCmd.Execute(); err != nil {
	//     fmt.Println(err)
	//     os.Exit(1)
	// }

	if err := fang.Execute(ctx, rootCmd); err != nil {
		os.Exit(1)
	}
}
