package cli

import (
	"fmt"
	"strings"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execution",
		Short: "Manage executions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List executions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			resp, err := client.ListExecutions(cmd.Context())
			if err != nil {
				return err
			}
			return printExecutionList(cmd, resp.Items)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get an execution by ID",
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
			execution, err := client.GetExecution(cmd.Context(), id)
			if err != nil {
				return err
			}
			return writeOutput(cmd, execution)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "logs <id>",
		Short: "Stream logs for an execution",
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
			chunks, err := client.GetExecutionLogs(cmd.Context(), id)
			if err != nil {
				return err
			}
			for _, c := range chunks {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s:%d] %s", c.Stream, c.Sequence, string(c.Payload))
				if !strings.HasSuffix(string(c.Payload), "\n") {
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cancel <id> [reason]",
		Short: "Request cancellation of an execution",
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
			req := contracts.CancelExecutionRequest{}
			if len(args) > 1 {
				req.Reason = args[1]
			}
			if err := client.CancelExecution(cmd.Context(), id, req); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Execution cancellation requested")
			return nil
		},
	})

	return cmd
}

func printExecutionList(cmd *cobra.Command, items []contracts.Execution) error {
	if viper.GetString("output") == "json" {
		return writeOutput(cmd, items)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-36s %-12s %-20s\n", "ID", "ASSIGNMENT", "STATE", "CREATED AT")
	for _, e := range items {
		fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-36s %-12s %-20s\n", e.ID, e.AssignmentID, e.State, e.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}
