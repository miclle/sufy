/*
 * Copyright (c) 2026 The SUFY Authors (sufy.com). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/sufy-dev/sufy/cmd/sufy/internal/sandbox/templatecfg"
	sdk "github.com/sufy-dev/sufy/sandbox"
)

// lookupTemplateIDByName looks up a template by its name (alias) in the
// current environment and returns its template_id.
//
// The backend guarantees alias uniqueness within an environment, so this
// is a stable locator key.
//
// Return value contract:
//   - Found: returns template_id, nil
//   - Not found: returns "", nil (caller decides whether to fall back to create)
//   - GetTemplateByAlias error: returns "", err
func lookupTemplateIDByName(ctx context.Context, client *sdk.Client, name string) (string, error) {
	if name == "" {
		return "", nil
	}
	tmpl, err := client.GetTemplateByAlias(ctx, name)
	if err != nil {
		if isTemplateAliasNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return tmpl.TemplateID, nil
}

// isTemplateAliasNotFound reports whether GetTemplateByAlias returned a 404.
func isTemplateAliasNotFound(err error) bool {
	var apiErr *sdk.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// templateIDFromCwdConfig loads sufy.sandbox.toml from the current working
// directory and returns a template_id usable for locating the template.
//
// Resolution order:
//  1. template_id from toml (backward compatible with existing configs)
//  2. name from toml → GetTemplateByAlias lookup
//
// This lets a toml that only declares name (no template_id) drive
// publish/get/delete commands, which makes it easy to share a single
// config across environments.
//
// On load/parse failure or remote-call failure, prints the error and
// returns ("", false). When the config file is absent, or has neither
// template_id nor a resolvable name, returns ("", true) so the caller
// can decide what to do next.
func templateIDFromCwdConfig() (string, bool) {
	cfg, err := templatecfg.LoadFromCwd()
	if err != nil {
		PrintError("load config: %v", err)
		return "", false
	}
	if cfg == nil {
		return "", true
	}
	if cfg.TemplateID != "" {
		fmt.Fprintf(os.Stderr, "[config] using template_id from %s\n", cfg.SourcePath())
		return cfg.TemplateID, true
	}
	if cfg.Name == "" {
		return "", true
	}

	client, cErr := NewSandboxClient()
	if cErr != nil {
		PrintError("%v", cErr)
		return "", false
	}
	id, lErr := lookupTemplateIDByName(context.Background(), client, cfg.Name)
	if lErr != nil {
		PrintError("lookup template by name %q: %v", cfg.Name, lErr)
		return "", false
	}
	if id == "" {
		return "", true
	}
	fmt.Fprintf(os.Stderr, "[lookup] template %q resolved to %s (from %s)\n",
		cfg.Name, id, cfg.SourcePath())
	return id, true
}
