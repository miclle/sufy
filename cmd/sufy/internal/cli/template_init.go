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

package cli

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"text/template"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

//go:embed templates/go/*.tmpl templates/typescript/*.tmpl templates/python/*.tmpl
var templateFS embed.FS

// validNamePattern validates template names: lowercase alphanumeric, starting with a-z or 0-9.
var validNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// supportedLanguages are the languages supported by the init scaffolding.
var supportedLanguages = []string{"go", "typescript", "python"}

// languageFiles maps language names to their template and output file pairs.
var languageFiles = map[string][]struct {
	tmpl   string // template file path within embed FS
	output string // output file name
}{
	"go": {
		{tmpl: "templates/go/main.go.tmpl", output: "main.go"},
		{tmpl: "templates/go/go.mod.tmpl", output: "go.mod"},
		{tmpl: "templates/go/Makefile.tmpl", output: "Makefile"},
	},
	"typescript": {
		{tmpl: "templates/typescript/template.ts.tmpl", output: "template.ts"},
		{tmpl: "templates/typescript/package.json.tmpl", output: "package.json"},
	},
	"python": {
		{tmpl: "templates/python/template.py.tmpl", output: "template.py"},
		{tmpl: "templates/python/requirements.txt.tmpl", output: "requirements.txt"},
	},
}

// InitInfo holds parameters for initializing a template project.
type InitInfo struct {
	Name     string // template project name
	Language string // programming language
	Path     string // output directory (defaults to ./<name>)
}

// TemplateInit initializes a new template project with scaffolded files.
// When parameters are not provided, uses interactive prompts.
func TemplateInit(info InitInfo) {
	name := info.Name
	language := info.Language
	path := info.Path

	// Interactive prompts if args are missing.
	if name == "" || language == "" {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			PrintError("--name and --language are required in non-interactive mode")
			return
		}

		var fields []huh.Field

		if name == "" {
			fields = append(fields,
				huh.NewInput().
					Title("Template name").
					Description("Lowercase alphanumeric, hyphens and underscores allowed").
					Value(&name).
					Validate(func(s string) error {
						if !validNamePattern.MatchString(s) {
							return fmt.Errorf("name must match pattern: [a-z0-9][a-z0-9_-]*")
						}
						return nil
					}),
			)
		}

		if language == "" {
			langOptions := make([]huh.Option[string], 0, len(supportedLanguages))
			for _, lang := range supportedLanguages {
				langOptions = append(langOptions, huh.NewOption(lang, lang))
			}
			fields = append(fields,
				huh.NewSelect[string]().
					Title("Programming language").
					Options(langOptions...).
					Value(&language),
			)
		}

		if len(fields) > 0 {
			form := huh.NewForm(huh.NewGroup(fields...))
			if err := form.Run(); err != nil {
				PrintError("cancelled: %v", err)
				return
			}
		}
	}

	// Validate name.
	if !validNamePattern.MatchString(name) {
		PrintError("invalid template name %q (must match: [a-z0-9][a-z0-9_-]*)", name)
		return
	}

	// Validate language.
	if !slices.Contains(supportedLanguages, language) {
		PrintError("unsupported language %q (supported: go, typescript, python)", language)
		return
	}

	if path == "" {
		path = "./" + name
	}

	fmt.Printf("Initializing %s template %q in %s...\n", language, name, path)
	if err := scaffold(name, language, path); err != nil {
		PrintError("scaffold failed: %v", err)
		return
	}
	PrintSuccess("Template %s initialized successfully!", name)
}

// scaffoldData holds template rendering context.
type scaffoldData struct {
	Name string
}

// scaffold generates project files for the given language in the target directory.
func scaffold(name, language, targetDir string) error {
	files, ok := languageFiles[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}

	data := scaffoldData{Name: name}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	for _, f := range files {
		tmplContent, err := templateFS.ReadFile(f.tmpl)
		if err != nil {
			return fmt.Errorf("read template %s: %w", f.tmpl, err)
		}

		tmpl, err := template.New(f.output).Parse(string(tmplContent))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", f.tmpl, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute template %s: %w", f.tmpl, err)
		}

		outPath := filepath.Join(targetDir, f.output)
		if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		PrintSuccess("  Created %s", outPath)
	}

	return nil
}
