package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Omotolani98/monocron/internal/cli"
	"github.com/Omotolani98/monocron/internal/cli/commands"
)

func main() {
	var output string
	flag.StringVar(&output, "output", "table", "Output format: table or json")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	if err := loadConfigIntoEnv(); err != nil {
		fmt.Fprintln(os.Stderr, "Warning:", err)
	}

	ctx := context.Background()
	var err error
	switch args[0] {
	case "login":
		err = commands.RunLogin(args[1:])
	case "schedule":
		err = commands.RunSchedule(ctx, args[1:], output)
	case "runner":
		err = commands.RunRunner(ctx, args[1:], output)
	case "execution":
		err = commands.RunExecution(ctx, args[1:], output)
	case "audit":
		err = commands.RunAudit(ctx, args[1:], output)
	case "init":
		err = commands.RunInit(ctx)
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`monocronctl - Monocron control CLI

Usage:
  monocronctl login <controller-url> <api-key>
  monocronctl schedule create <name> <cron> <timeout> <command> [label=value ...]
  monocronctl schedule list
  monocronctl schedule get <id>
  monocronctl schedule update <id> <field=value> ...
  monocronctl schedule remove <id>
  monocronctl runner token [label=value ...]
  monocronctl runner join <token>
  monocronctl runner list
  monocronctl runner get <id>
  monocronctl runner drain <id> [reason]
  monocronctl runner remove <id>
  monocronctl execution list
  monocronctl execution get <id>
  monocronctl execution logs <id>
  monocronctl execution cancel <id> [reason]
  monocronctl audit list
  monocronctl init

Global flags:
  -output table|json
`)
}

func loadConfigIntoEnv() error {
	cfg, err := cli.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.ControllerURL != "" && os.Getenv("MONOCRON_CONTROLLER_URL") == "" {
		os.Setenv("MONOCRON_CONTROLLER_URL", cfg.ControllerURL)
	}
	if cfg.APIKey != "" && os.Getenv("MONOCRON_API_KEY") == "" {
		os.Setenv("MONOCRON_API_KEY", cfg.APIKey)
	}
	return nil
}
