package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var runnersCmd = &cobra.Command{
	Use:   "runners",
	Short: "Manage runners",
}

var runnersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List runners",
	RunE: func(cmd *cobra.Command, args []string) error {
		var out struct {
			Runners []struct {
				RunnerID string    `json:"runner_id"`
				Kind     string    `json:"kind"`
				Status   string    `json:"status"`
				LastSeen time.Time `json:"last_seen"`
			} `json:"runners"`
		}
		if err := getJSON(apiBase+"/runners", &out); err != nil {
			return err
		}
		header := lipgloss.NewStyle().Bold(true)
		fmt.Printf("%s  %s  %s  %s\n", header.Render("runner_id"), header.Render("type"), header.Render("status"), header.Render("last_seen"))
		for _, r := range out.Runners {
			fmt.Printf("%-38s  %-10s  %-6s  %-25s\n", r.RunnerID, r.Kind, r.Status, r.LastSeen.Format(time.RFC3339))
		}
		return nil
	},
}

var runnersRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a runner",
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, _ := cmd.Flags().GetString("kind")
		if kind == "" {
			return fmt.Errorf("--kind required (vm|docker|bare-metal)")
		}
		var res struct {
			RunnerID string `json:"runner_id"`
		}
		in := struct {
			Kind string `json:"kind"`
		}{Kind: kind}
		if err := postJSON(apiBase+"/runners/register", &in, &res); err != nil {
			return err
		}
		fmt.Println("runner_id:", res.RunnerID)
		return nil
	},
}

var runnersStopCmd = &cobra.Command{
	Use:   "stop <runner_id>",
	Short: "Stop a runner",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		in := struct {
			RunnerID string `json:"runner_id"`
		}{RunnerID: args[0]}

		var res struct{}
		if err := postJSON(apiBase+"/runners/stop", &in, &res); err != nil {
			return err
		}
		fmt.Println("stop signal sent to", args[0])
		return nil
	},
}

var runnersKillCmd = &cobra.Command{
	Use:   "kill <runner_id>",
	Short: "Kill a runner",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		in := struct {
			RunnerID string `json:"runner_id"`
		}{RunnerID: args[0]}

		var res struct{}
		if err := postJSON(apiBase+"/runners/kill", &in, &res); err != nil {
			return err
		}
		fmt.Println("kill signal sent to", args[0])
		return nil
	},
}

func init() {
	runnersCmd.AddCommand(runnersListCmd)
	runnersRegisterCmd.Flags().String("kind", "", "Runner kind (vm|docker|bare-metal)")
	runnersCmd.AddCommand(runnersRegisterCmd)
	runnersCmd.AddCommand(runnersStopCmd)
	runnersCmd.AddCommand(runnersKillCmd)
}
