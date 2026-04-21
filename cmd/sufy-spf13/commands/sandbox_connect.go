package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxConnectCmd corresponds to cmd/sufy/sandbox_connect_cmd.gox.
var sandboxConnectCmd = &cobra.Command{
	Use:     "connect <sandboxID>",
	Short:   "Connect to an existing sandbox terminal (alias: cn)",
	Aliases: []string{"cn"},
	Example: `  # Connect to a sandbox by ID
  sufy sandbox connect sb-xxxxxxxxxxxx
  sufy sbx cn sb-xxxxxxxxxxxx`,
	Run: func(_ *cobra.Command, args []string) {
		if len(args) == 0 {
			sandbox.Connect("")
			return
		}
		sandbox.Connect(args[0])
	},
}

func init() {
	sandboxCmd.AddCommand(sandboxConnectCmd)
}
