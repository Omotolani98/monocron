package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit logs",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List audit events",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			resp, err := client.ListAuditEvents(cmd.Context())
			if err != nil {
				return err
			}
			if viper.GetString("output") == "json" {
				return writeOutput(cmd, resp.Items)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-12s %-12s %-20s %-20s\n", "ID", "ACTOR", "ACTION", "TARGET", "CREATED AT")
			for _, e := range resp.Items {
				fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-12s %-12s %-20s %-20s\n", e.ID, e.Actor, e.Action, e.Target, e.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	})

	return cmd
}
