package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/fang/v2"
	"github.com/Omotolani98/monocron/internal/platform/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand returns the Cobra root command for monocronctl.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "monocronctl",
		Short: "Control CLI for Monocron",
		Long:  "monocronctl manages Monocron schedules, runners, executions, and audit logs.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().String("config", "", "config file path (default is platform config dir)")
	root.PersistentFlags().String("controller-url", "", "controller base URL")
	root.PersistentFlags().String("api-key", "", "API key for controller authentication")
	root.PersistentFlags().StringP("output", "o", "table", "output format: table or json")
	root.PersistentFlags().Duration("request-timeout", 30*time.Second, "HTTP request timeout")

	_ = viper.BindPFlag("controller_url", root.PersistentFlags().Lookup("controller-url"))
	_ = viper.BindPFlag("api_key", root.PersistentFlags().Lookup("api-key"))
	_ = viper.BindPFlag("output", root.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("request_timeout", root.PersistentFlags().Lookup("request-timeout"))

	viper.SetEnvPrefix("MONOCRON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newScheduleCmd())
	root.AddCommand(newRunnerCmd())
	root.AddCommand(newExecutionCmd())
	root.AddCommand(newAuditCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newInitCmd())

	return root
}

// Execute runs the CLI through Fang with styled help, errors, completions, and version output.
func Execute() {
	root := NewRootCommand()
	if err := fang.Execute(
		root.Context(),
		root,
		fang.WithVersion(version.Version),
		fang.WithCommit(version.Commit),
	); err != nil {
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command) error {
	if skipConfigInit(cmd) {
		return nil
	}
	path, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	viper.SetConfigFile(path)
	viper.SetConfigType("json")
	if err := CreateDefaultIfMissing(path); err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return nil
}

func resolveConfigPath(cmd *cobra.Command) (string, error) {
	if f := cmd.Flag("config"); f != nil && f.Changed {
		return f.Value.String(), nil
	}
	if v := os.Getenv("MONOCRON_CONFIG"); v != "" {
		return v, nil
	}
	return DefaultConfigPath(), nil
}

func skipConfigInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "completion", "man":
			return true
		}
	}
	if cmd.Flags().Changed("version") {
		return true
	}
	return false
}

func clientFromCmd(cmd *cobra.Command) (*CLIClient, error) {
	baseURL := viper.GetString("controller_url")
	apiKey := viper.GetString("api_key")
	timeout := viper.GetDuration("request_timeout")
	if baseURL == "" {
		return nil, fmt.Errorf("controller URL is required; run 'monocronctl login' or set --controller-url/MONOCRON_CONTROLLER_URL")
	}
	return NewCLIClient(baseURL, apiKey, timeout), nil
}
