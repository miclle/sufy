// Package main provides an alternative CLI entry point built on
// github.com/spf13/cobra, mirroring the behavior of ./cmd/sufy (which uses
// github.com/goplus/cobra via .gox files). It shares the same underlying
// business logic through cmd/internal/sandbox so the only variable is the
// CLI framework itself.
package main

import (
	"fmt"
	"os"

	"github.com/sufy-dev/sufy/cmd/sufy-spf13/commands"
)

func main() {
	if err := commands.Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
