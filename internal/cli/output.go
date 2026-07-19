package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// writeOutput prints v according to the configured output format.
func writeOutput(cmd *cobra.Command, v any) error {
	output := viper.GetString("output")
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if output == "table" {
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

// writeRawOutput prints raw bytes without re-encoding.
func writeRawOutput(cmd *cobra.Command, data []byte) {
	fmt.Fprint(cmd.OutOrStdout(), string(data))
}
