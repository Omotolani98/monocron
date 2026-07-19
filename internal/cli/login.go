package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <controller-url> <api-key>",
		Short: "Save controller URL and API key to the config file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			cfg := Config{
				ControllerURL: args[0],
				APIKey:        args[1],
			}
			if err := WriteConfig(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
			return nil
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored API key from the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			cfg, err := ReadConfig(path)
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}
			cfg.APIKey = ""
			if err := WriteConfig(path, cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
			return nil
		},
	}
}
