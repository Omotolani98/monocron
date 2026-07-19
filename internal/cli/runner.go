package cli

import (
	"fmt"
	"os"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/runner/config"
	"github.com/Omotolani98/monocron/internal/runner/controllerclient"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRunnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Manage runners",
	}

	cmd.PersistentFlags().String("state", "", "runner state file path")

	cmd.AddCommand(&cobra.Command{
		Use:   "token [label=value ...]",
		Short: "Create an enrollment token for a new runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			req := contracts.CreateEnrollmentTokenRequest{Labels: parseLabels(args)}
			resp, err := client.CreateEnrollmentToken(cmd.Context(), req)
			if err != nil {
				return err
			}
			if viper.GetString("output") == "json" {
				return writeOutput(cmd, resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Enrollment token:", resp.Token)
			fmt.Fprintln(cmd.OutOrStdout(), "Expires at:", resp.ExpiresAt)
			return nil
		},
	})

	joinCmd := &cobra.Command{
		Use:   "join <token>",
		Short: "Enroll this machine as a runner and persist credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			controllerURL := viper.GetString("controller_url")
			if controllerURL == "" {
				return fmt.Errorf("controller URL is required; run 'monocronctl login' or set --controller-url/MONOCRON_CONTROLLER_URL")
			}

			statePath, err := resolveStatePath(cmd)
			if err != nil {
				return err
			}

			store := controllerclient.NewTokenStore(statePath)
			if !cmd.Flags().Changed("force") {
				if _, err := store.Load(); err == nil {
					return fmt.Errorf("runner state already exists at %s; use --force to overwrite", statePath)
				}
			}

			controller := controllerclient.New(controllerURL, "")
			resp, err := controller.Enroll(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("enroll: %w", err)
			}
			if err := store.Save(controllerclient.RunnerState{RunnerID: resp.RunnerID, AccessToken: resp.AccessToken}); err != nil {
				return fmt.Errorf("save runner state: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Runner joined:", resp.RunnerID)
			fmt.Fprintln(cmd.OutOrStdout(), "State saved to:", statePath)
			return nil
		},
	}
	joinCmd.Flags().Bool("force", false, "overwrite an existing runner state file")
	cmd.AddCommand(joinCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List registered runners",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			resp, err := client.ListRunners(cmd.Context())
			if err != nil {
				return err
			}
			return printRunnerList(cmd, resp.Items)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get a runner by ID",
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
			runner, err := client.GetRunner(cmd.Context(), id)
			if err != nil {
				return err
			}
			return writeOutput(cmd, runner)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "drain <id> [reason]",
		Short: "Drain a runner",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			id, err := uuid.Parse(args[0])
			if err != nil {
				return err
			}
			req := contracts.DrainRunnerRequest{}
			if len(args) > 1 {
				req.Reason = args[1]
			}
			if err := client.DrainRunner(cmd.Context(), id, req); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Runner draining")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "remove <id>",
		Short:   "Remove a runner",
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
			if err := client.DeleteRunner(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Runner removed")
			return nil
		},
	})

	return cmd
}

func printRunnerList(cmd *cobra.Command, items []contracts.Runner) error {
	if viper.GetString("output") == "json" {
		return writeOutput(cmd, items)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-10s %-12s %-20s\n", "ID", "TYPE", "STATE", "LAST HEARTBEAT")
	for _, r := range items {
		fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-10s %-12s %-20s\n", r.ID, r.Type, r.State, formatTime(&r.LastHeartbeat))
	}
	return nil
}

func resolveStatePath(cmd *cobra.Command) (string, error) {
	if f := cmd.Flag("state"); f != nil && f.Changed {
		return f.Value.String(), nil
	}
	if v := os.Getenv("MONOCRON_RUNNER_STATE_PATH"); v != "" {
		return v, nil
	}
	cfgPath := config.DefaultConfigPath()
	if cfg, err := config.ReadConfig(cfgPath); err == nil && cfg.StatePath != "" {
		return cfg.StatePath, nil
	}
	return config.DefaultStatePath(), nil
}
