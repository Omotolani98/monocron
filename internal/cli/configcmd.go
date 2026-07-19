package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the monocronctl configuration file",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create the config file if it does not already exist",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			if err := CreateDefaultIfMissing(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config ready at %s\n", path)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the resolved config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Display the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			if err := CreateDefaultIfMissing(path); err != nil {
				return err
			}
			cfg, err := ReadConfig(path)
			if err != nil {
				return err
			}
			cfg.APIKey = "***"
			return writeOutput(cmd, cfg)
		},
	})

	return cmd
}
