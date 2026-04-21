package commands

import "github.com/spf13/cobra"

// sandboxCmd corresponds to cmd/sufy/sandbox_cmd.gox — the `sandbox` command group.
var sandboxCmd = &cobra.Command{
	Use:     "sandbox",
	Short:   "Manage sandboxes (alias: sbx)",
	Aliases: []string{"sbx"},
	Example: `  # View sandbox subcommands
  sufy sandbox -h
  sufy sbx -h

  # Create a sandbox from a template
  sufy sandbox create my-template
  sufy sbx cr my-template

  # List running sandboxes
  sufy sandbox list
  sufy sbx ls

  # Connect to a sandbox
  sufy sandbox connect sb-xxxxxxxxxxxx
  sufy sbx cn sb-xxxxxxxxxxxx`,
	// When invoked without a subcommand, show help — same behavior as the
	// `run => { help }` block in the .gox source.
	Run: func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
}

func init() {
	Root.AddCommand(sandboxCmd)
}
