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
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
)

// Output format constants shared by commands with -f/--format flags.
const (
	FormatPretty = "pretty"
	FormatJSON   = "json"
)

// Color helpers for consistent CLI styling.
var (
	ColorError   = color.New(color.FgRed)
	ColorSuccess = color.New(color.FgGreen)
	ColorWarn    = color.New(color.FgYellow)
	ColorInfo    = color.New(color.FgCyan)
	ColorMuted   = color.New(color.FgHiBlack)
)

// PrintError writes a red "Error:"-prefixed message to stderr.
func PrintError(format string, a ...any) {
	ColorError.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
}

// PrintSuccess writes a green message to stdout.
func PrintSuccess(format string, a ...any) {
	ColorSuccess.Printf(format+"\n", a...)
}

// PrintWarn writes a yellow "Warning:"-prefixed message to stderr.
func PrintWarn(format string, a ...any) {
	ColorWarn.Fprintf(os.Stderr, "Warning: "+format+"\n", a...)
}

// logLevelColors maps log levels to their badge colors.
var logLevelColors = map[string]*color.Color{
	"debug": color.New(color.FgWhite),
	"info":  color.New(color.FgGreen),
	"warn":  color.New(color.FgYellow),
	"error": color.New(color.FgRed),
}

// LogLevelBadge returns a 5-column colorized log level label.
func LogLevelBadge(level string) string {
	lower := strings.ToLower(level)
	c, ok := logLevelColors[lower]
	if !ok {
		return fmt.Sprintf("%-5s", strings.ToUpper(level))
	}
	return c.Sprintf("%-5s", strings.ToUpper(level))
}

// NewTable creates a tabwriter suitable for aligned column output.
func NewTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// FormatTimestamp formats a time as RFC3339, or "-" for zero values.
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

// FormatBytes renders bytes as MiB to match conventional CLI output.
func FormatBytes(b int64) string {
	mib := float64(b) / (1024 * 1024)
	if mib == float64(int64(mib)) {
		return fmt.Sprintf("%d MiB", int64(mib))
	}
	return fmt.Sprintf("%.1f MiB", mib)
}

// FormatMetadata renders a metadata map as "k1=v1, k2=v2" or "-" when empty.
func FormatMetadata(m map[string]string) string {
	if len(m) == 0 {
		return "-"
	}
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ", ")
}

// FormatOptionalString returns *s or "-" when nil/empty.
func FormatOptionalString(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}
