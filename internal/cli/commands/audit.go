package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/Omotolani98/monocron/internal/cli"
)

// RunAudit lists audit events.
func RunAudit(ctx context.Context, args []string, output string) error {
	client := cli.NewCLIClient()
	resp, err := client.ListAuditEvents(ctx)
	if err != nil {
		return err
	}
	if output == "json" {
		return writeOutput("json", resp.Items)
	}
	fmt.Fprintf(os.Stdout, "%-36s %-12s %-12s %-20s %-20s\n", "ID", "ACTOR", "ACTION", "TARGET", "CREATED AT")
	for _, e := range resp.Items {
		fmt.Fprintf(os.Stdout, "%-36s %-12s %-12s %-20s %-20s\n", e.ID, e.Actor, e.Action, e.Target, e.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}
