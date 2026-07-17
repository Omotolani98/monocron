package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Omotolani98/monocron/internal/cli"
	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

// RunSchedule is the entry point for monocronctl schedule commands.
func RunSchedule(ctx context.Context, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl schedule <create|list|get|update|remove> ...")
	}
	client := cli.NewCLIClient()
	switch args[0] {
	case "create":
		return runScheduleCreate(ctx, client, args[1:], output)
	case "list":
		return runScheduleList(ctx, client, output)
	case "get":
		return runScheduleGet(ctx, client, args[1:], output)
	case "update":
		return runScheduleUpdate(ctx, client, args[1:], output)
	case "remove":
		return runScheduleRemove(ctx, client, args[1:])
	default:
		return fmt.Errorf("unknown schedule subcommand: %s", args[0])
	}
}

func runScheduleCreate(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: monocronctl schedule create <name> <cron> <timeout> <command> [label=value ...]")
	}
	labels := parseLabels(args[3:])
	command := strings.Split(args[2], " ")
	req := contracts.CreateScheduleRequest{
		Name:    args[0],
		Cron:    args[1],
		Timeout: args[2],
		Command: command,
		Labels:  labels,
	}
	// Override command with joined remaining args if labels not present
	// Simplification: treat arg[2] as timeout and use rest as command until labels
	req.Command, req.Labels = splitCommandAndLabels(args[2:])
	req.Timeout = args[2]

	spec, err := client.CreateSchedule(ctx, req)
	if err != nil {
		return err
	}
	return writeOutput(output, spec)
}

func runScheduleList(ctx context.Context, client *cli.CLIClient, output string) error {
	resp, err := client.ListSchedules(ctx)
	if err != nil {
		return err
	}
	if output == "json" {
		return writeOutput("json", resp.Items)
	}
	fmt.Fprintf(os.Stdout, "%-36s %-20s %-20s %-10s\n", "ID", "NAME", "CRON", "ENABLED")
	for _, s := range resp.Items {
		fmt.Fprintf(os.Stdout, "%-36s %-20s %-20s %-10v\n", s.ID, s.Name, s.Cron, s.Enabled)
	}
	return nil
}

func runScheduleGet(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl schedule get <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	spec, err := client.GetSchedule(ctx, id)
	if err != nil {
		return err
	}
	return writeOutput(output, spec)
}

func runScheduleUpdate(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: monocronctl schedule update <id> <field=value> ...")
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
	spec, err := client.UpdateSchedule(ctx, id, req)
	if err != nil {
		return err
	}
	return writeOutput(output, spec)
}

func runScheduleRemove(ctx context.Context, client *cli.CLIClient, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl schedule remove <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	if err := client.DeleteSchedule(ctx, id); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Schedule removed")
	return nil
}

func splitCommandAndLabels(args []string) ([]string, contracts.Labels) {
	labels := contracts.Labels{}
	var command []string
	for _, a := range args {
		if strings.Contains(a, "=") {
			parts := strings.SplitN(a, "=", 2)
			labels[parts[0]] = parts[1]
		} else {
			command = append(command, a)
		}
	}
	return command, labels
}

func parseLabels(args []string) contracts.Labels {
	labels := contracts.Labels{}
	for _, a := range args {
		if strings.Contains(a, "=") {
			parts := strings.SplitN(a, "=", 2)
			labels[parts[0]] = parts[1]
		}
	}
	return labels
}

func writeOutput(output string, v any) error {
	if output == "json" {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
	return nil
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
}
