// Package commands defines the spf13/cobra command tree for the sufy CLI.
//
// Structure mirrors cmd/sufy/*.gox one-to-one: each .gox file corresponds
// to a file here. The command tree is assembled in init() via AddCommand.
package commands

import "github.com/spf13/cobra"

// Root is the top-level `sufy` command.
var Root = &cobra.Command{
	Use:   "sufy",
	Short: "sufy - A unified tool to manage your SUFY cloud services",
	// Silence cobra's automatic usage/error printing on runtime errors; the
	// underlying business layer prints its own diagnostics.
	SilenceUsage:  true,
	SilenceErrors: true,
}
