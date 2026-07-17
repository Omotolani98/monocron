package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/Omotolani98/monocron/internal/cli"
	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/runner/controllerclient"
	"github.com/google/uuid"
)

// RunRunner is the entry point for monocronctl runner commands.
func RunRunner(ctx context.Context, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl runner <token|join|list|get|drain|remove> ...")
	}
	client := cli.NewCLIClient()
	switch args[0] {
	case "token":
		return runRunnerToken(ctx, client, args[1:], output)
	case "join":
		return runRunnerJoin(ctx, args[1:])
	case "list":
		return runRunnerList(ctx, client, output)
	case "get":
		return runRunnerGet(ctx, client, args[1:], output)
	case "drain":
		return runRunnerDrain(ctx, client, args[1:])
	case "remove":
		return runRunnerRemove(ctx, client, args[1:])
	default:
		return fmt.Errorf("unknown runner subcommand: %s", args[0])
	}
}

func runRunnerToken(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	req := contracts.CreateEnrollmentTokenRequest{}
	for _, a := range args {
		if contains(a, "=") {
			parts := split2(a, "=")
			if req.Labels == nil {
				req.Labels = contracts.Labels{}
			}
			req.Labels[parts[0]] = parts[1]
		}
	}
	resp, err := client.CreateEnrollmentToken(ctx, req)
	if err != nil {
		return err
	}
	if output == "json" {
		return writeOutput("json", resp)
	}
	fmt.Fprintln(os.Stdout, "Enrollment token:", resp.Token)
	fmt.Fprintln(os.Stdout, "Expires at:", resp.ExpiresAt)
	return nil
}

func runRunnerJoin(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl runner join <token>")
	}
	cfg, err := cli.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w; run monocronctl login first", err)
	}
	controller := controllerclient.New(cfg.ControllerURL, "")
	resp, err := controller.Enroll(ctx, args[0])
	if err != nil {
		return fmt.Errorf("enroll: %w", err)
	}
	store := controllerclient.NewTokenStore(os.Getenv("MONOCRON_RUNNER_STATE_PATH"))
	if store.Path == "" {
		store = controllerclient.NewTokenStore("/var/lib/monocron/runner.state")
	}
	if err := store.Save(controllerclient.RunnerState{RunnerID: resp.RunnerID, AccessToken: resp.AccessToken}); err != nil {
		return fmt.Errorf("save runner state: %w", err)
	}
	fmt.Fprintln(os.Stdout, "Runner joined:", resp.RunnerID)
	fmt.Fprintln(os.Stdout, "State saved to:", store.Path)
	return nil
}

func runRunnerList(ctx context.Context, client *cli.CLIClient, output string) error {
	resp, err := client.ListRunners(ctx)
	if err != nil {
		return err
	}
	if output == "json" {
		return writeOutput("json", resp.Items)
	}
	fmt.Fprintf(os.Stdout, "%-36s %-10s %-12s %-20s\n", "ID", "TYPE", "STATE", "LAST HEARTBEAT")
	for _, r := range resp.Items {
		fmt.Fprintf(os.Stdout, "%-36s %-10s %-12s %-20s\n", r.ID, r.Type, r.State, formatTime(&r.LastHeartbeat))
	}
	return nil
}

func runRunnerGet(ctx context.Context, client *cli.CLIClient, args []string, output string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl runner get <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	runner, err := client.GetRunner(ctx, id)
	if err != nil {
		return err
	}
	return writeOutput(output, runner)
}

func runRunnerDrain(ctx context.Context, client *cli.CLIClient, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl runner drain <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	req := contracts.DrainRunnerRequest{}
	if len(args) > 1 {
		req.Reason = args[1]
	}
	if err := client.DrainRunner(ctx, id, req); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Runner draining")
	return nil
}

func runRunnerRemove(ctx context.Context, client *cli.CLIClient, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: monocronctl runner remove <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return err
	}
	if err := client.DeleteRunner(ctx, id); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Runner removed")
	return nil
}

func contains(s, substr string) bool { return len(split2(s, substr)) > 1 }

func split2(s, sep string) []string {
	idx := -1
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}
