package commands

import (
	"github.com/spf13/cobra"

	"github.com/sufy-dev/sufy/cmd/internal/sandbox"
)

// sandboxCreateCmd corresponds to cmd/sufy/sandbox_create_cmd.gox.
var sandboxCreateCmd = func() *cobra.Command {
	var (
		metadata         string
		autoPause        bool
		detach           bool
		timeout          int32
		envVars          []string
		injectionRuleIDs []string
		inlineInjections []string
	)
	cmd := &cobra.Command{
		Use:     "create [template]",
		Short:   "Create a sandbox and connect to its terminal (alias: cr)",
		Aliases: []string{"cr"},
		Args:    cobra.MaximumNArgs(1),
		Example: `  # Create a sandbox from a template
  sufy sandbox create my-template
  sufy sbx cr my-template

  # Create with a timeout (seconds)
  sufy sandbox create my-template --timeout 300
  sufy sbx cr my-template -t 300

  # Create in detached mode (no terminal, sandbox stays alive)
  sufy sandbox create my-template -t 300 --detach
  sufy sbx cr my-template -t 300 --detach

  # Create with environment variables
  sufy sandbox create my-template -e FOO=bar -e BAZ=qux
  sufy sbx cr my-template -e FOO=bar -e BAZ=qux

  # Create with auto-pause (pause instead of kill on timeout)
  sufy sandbox create my-template -t 300 --auto-pause
  sufy sbx cr my-template -t 300 --auto-pause

  # Create with metadata
  sufy sandbox create my-template -m env=dev,team=backend
  sufy sbx cr my-template -m env=dev,team=backend

  # Create with injection rules
  sufy sandbox create my-template --injection-rule rule-openai --injection-rule rule-http
  sufy sbx cr my-template --injection-rule rule-openai --injection-rule rule-http

  # Create with inline injections
  sufy sandbox create my-template --inline-injection 'type=openai,api-key=sk-xxx' --inline-injection 'type=http,base-url=https://api.example.com,headers=Authorization=Bearer token;X-Env=prod'
  sufy sbx cr my-template --inline-injection 'type=openai,api-key=sk-xxx'`,
		Run: func(_ *cobra.Command, args []string) {
			templateID := ""
			if len(args) > 0 {
				templateID = args[0]
			}
			sandbox.Create(sandbox.CreateInfo{
				TemplateID:       templateID,
				Timeout:          timeout,
				Metadata:         metadata,
				Detach:           detach,
				EnvVars:          envVars,
				AutoPause:        autoPause,
				InjectionRuleIDs: injectionRuleIDs,
				InlineInjections: inlineInjections,
			})
		},
	}
	f := cmd.Flags()
	f.StringVarP(&metadata, "metadata", "m", "", "metadata key=value pairs (comma-separated)")
	f.BoolVar(&autoPause, "auto-pause", false, "automatically pause sandbox when timeout expires (instead of killing)")
	f.BoolVar(&detach, "detach", false, "create sandbox without connecting terminal (sandbox stays alive until timeout)")
	f.Int32VarP(&timeout, "timeout", "t", 0, "sandbox timeout in seconds")
	f.StringArrayVarP(&envVars, "env-var", "e", nil, "environment variables (KEY=VALUE, can be specified multiple times)")
	f.StringArrayVar(&injectionRuleIDs, "injection-rule", nil, "injection rule IDs to apply when creating the sandbox (can be specified multiple times)")
	f.StringArrayVar(&inlineInjections, "inline-injection", nil, "inline injection spec to apply when creating the sandbox (can be specified multiple times, format: type=<type>,api-key=<key>,base-url=<url>,headers=<k1=v1;k2=v2>)")
	return cmd
}()

func init() {
	sandboxCmd.AddCommand(sandboxCreateCmd)
}
