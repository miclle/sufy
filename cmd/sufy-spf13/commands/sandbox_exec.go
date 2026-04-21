package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxExecCmd corresponds to cmd/sufy/sandbox_exec_cmd.gox.
var sandboxExecCmd = func() *cobra.Command {
	var (
		background bool
		cwd        string
		user       string
		envVars    []string
	)
	cmd := &cobra.Command{
		Use:     "exec <sandboxID> -- <command...>",
		Short:   "Execute a command in a sandbox (alias: ex)",
		Aliases: []string{"ex"},
		Args:    cobra.MinimumNArgs(1),
		Example: `  # Run a command in a sandbox
  sufy sandbox exec sb-xxxxxxxxxxxx -- ls -la
  sufy sbx ex sb-xxxxxxxxxxxx -- ls -la

  # Pipe stdin to a command
  echo "hello world" | sufy sbx ex sb-xxxxxxxxxxxx -- cat
  cat file.txt | sufy sbx ex sb-xxxxxxxxxxxx -- wc -l

  # Run in background (print PID and return)
  sufy sandbox exec sb-xxxxxxxxxxxx -b -- python server.py
  sufy sbx ex sb-xxxxxxxxxxxx -b -- python server.py

  # Specify working directory and user
  sufy sandbox exec sb-xxxxxxxxxxxx -c /app -u root -- npm install

  # Set environment variables
  sufy sandbox exec sb-xxxxxxxxxxxx -e PORT=3000 -e NODE_ENV=production -- node app.js`,
		Run: func(_ *cobra.Command, args []string) {
			sandbox.Exec(sandbox.ExecInfo{
				SandboxID:  args[0],
				Command:    args[1:],
				Background: background,
				Cwd:        cwd,
				User:       user,
				EnvVars:    envVars,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&background, "background", "b", false, "run command in background (print PID and return)")
	f.StringVarP(&cwd, "cwd", "c", "", "working directory for the command")
	f.StringVarP(&user, "user", "u", "", "user to run the command as")
	f.StringArrayVarP(&envVars, "env", "e", nil, "environment variables (KEY=VALUE, can be specified multiple times)")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxExecCmd)
}
