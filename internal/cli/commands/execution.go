package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Omotolani98/monocron/internal/cli"
	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

// RunExecution is the entry point for monocronctl execution commands.
func RunExecution(ctx context.Context, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl execution <list|get|logs|cancel> ...")
	}
	client := cli.NewCLIClient()
	switch args[0] {
	case "list":
		return runExecutionList(ctx, client, output)
	case "get":
		return runExecutionGet(ctx, client, args[1:], output)
	case "logs":
		return runExecutionLogs(ctx, client, args[1:])
	case "cancel":
		return runExecutionCancel(ctx, client, args[1:])
	default:
		return fmt.Errorf("unknown execution subcommand: %s", args[0])
	}
}

func runExecutionList(ctx context.Context, client *cli.CLIClient, output string) error {
	resp, err := client.ListExecutions(ctx)
	if err != nil {
		return err
	}
	if output == "json" {
		return writeOutput("json", resp.Items)
	}
	fmt.Fprintf(os.Stdout, "%-36s %-36s %-12s %-20s\n", "ID", "ASSIGNMENT", "STATE", "CREATED AT")
	for _, e := range resp.Items {
		fmt.Fprintf(os.Stdout, "%-36s %-36s %-12s %-20s\n", e.ID, e.AssignmentID, e.State, e.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runExecutionGet(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl execution get <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	execution, err := client.GetExecution(ctx, id)
	if err != nil {
		return err
	}
	return writeOutput(output, execution)
}

func runExecutionLogs(ctx context.Context, client *cli.CLIClient, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl execution logs <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	chunks, err := client.GetExecutionLogs(ctx, id)
	if err != nil {
		return err
	}
	for _, c := range chunks {
		fmt.Fprintf(os.Stdout, "[%s:%d] %s", c.Stream, c.Sequence, string(c.Payload))
		if !strings.HasSuffix(string(c.Payload), "\n") {
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}

func runExecutionCancel(ctx context.Context, client *cli.CLIClient, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl execution cancel <id> [reason]")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	req := contracts.CancelExecutionRequest{}
	if len(args) > 1 {
		req.Reason = args[1]
	}
	if err := client.CancelExecution(ctx, id, req); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Execution cancellation requested")
	return nil
}
