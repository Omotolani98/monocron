package cli

import (
	"fmt"
	"strings"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage schedules",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create <name> <cron> <timeout> <command> [label=value ...]",
		Short: "Create a new schedule",
		Args:  cobra.MinimumNArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			command, labels := splitCommandAndLabels(args[3:])
			req := contracts.CreateScheduleRequest{
				Name:    args[0],
				Cron:    args[1],
				Timeout: args[2],
				Command: command,
				Labels:  labels,
			}
			spec, err := client.CreateSchedule(cmd.Context(), req)
			if err != nil {
				return err
			}
			return writeOutput(cmd, spec)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			resp, err := client.ListSchedules(cmd.Context())
			if err != nil {
				return err
			}
			return printScheduleList(cmd, resp.Items)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get a schedule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			id, err := uuid.Parse(args[0])
			if err != nil {
				return err
			}
			spec, err := client.GetSchedule(cmd.Context(), id)
			if err != nil {
				return err
			}
			return writeOutput(cmd, spec)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "update <id> <field=value> ...",
		Short: "Update a schedule",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			id, err := uuid.Parse(args[0])
			if err != nil {
				return err
			}
			req := contracts.UpdateScheduleRequest{}
			for _, kv := range args[1:] {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid field %q", kv)
				}
				switch parts[0] {
				case "name":
					req.Name = parts[1]
				case "cron":
					req.Cron = parts[1]
				case "timeout":
					req.Timeout = parts[1]
				case "command":
					req.Command = strings.Split(parts[1], " ")
				case "enabled":
					b := parts[1] == "true"
					req.Enabled = &b
				default:
					if req.Labels == nil {
						req.Labels = contracts.Labels{}
					}
					req.Labels[parts[0]] = parts[1]
				}
			}
			spec, err := client.UpdateSchedule(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			return writeOutput(cmd, spec)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "remove <id>",
		Short:   "Remove a schedule",
		Aliases: []string{"delete", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			id, err := uuid.Parse(args[0])
			if err != nil {
				return err
			}
			if err := client.DeleteSchedule(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Schedule removed")
			return nil
		},
	})

	return cmd
}

func printScheduleList(cmd *cobra.Command, items []contracts.ScheduleSpec) error {
	if viper.GetString("output") == "json" {
		return writeOutput(cmd, items)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-20s %-20s %-10s\n", "ID", "NAME", "CRON", "ENABLED")
	for _, s := range items {
		fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-20s %-20s %-10v\n", s.ID, s.Name, s.Cron, s.Enabled)
	}
	return nil
}

// splitCommandAndLabels splits args into command arguments and labels.
// Labels are key=value pairs; the command consists of all arguments before the first label.
func splitCommandAndLabels(args []string) ([]string, contracts.Labels) {
	labels := contracts.Labels{}
	var split int
	for i, a := range args {
		if strings.Contains(a, "=") {
			split = i
			break
		}
	}
	command := args[:split]
	for _, a := range args[split:] {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	return command, labels
}

func parseLabels(args []string) contracts.Labels {
	labels := contracts.Labels{}
	for _, a := range args {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	return labels
}

