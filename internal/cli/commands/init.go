package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/Omotolani98/monocron/internal/cli"
)

// RunLogin prompts for controller URL and API key and saves configuration.
func RunLogin(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: monocronctl login <controller-url> <api-key>")
	}
	cfg := cli.Config{
		ControllerURL: args[0],
		APIKey:        args[1],
	}
	if err := cli.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintln(os.Stdout, "Configuration saved to", cli.ConfigPath())
	return nil
}

// RunInit installs and configures monocrond and monocron-runner.
func RunInit(ctx context.Context) error {
	fmt.Fprintln(os.Stdout, "monocronctl init is not yet implemented")
	return nil
}
