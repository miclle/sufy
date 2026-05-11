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

package templatecfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load reads and parses the TOML config at the given path.
//
// Returns (nil, nil) when the file does not exist; returns an error on
// I/O failure or parse failure.
func Load(path string) (*FileConfig, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config %s: %w", abs, err)
	}

	cfg := &FileConfig{}
	md, err := toml.NewDecoder(bytes.NewReader(data)).Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", abs, err)
	}
	cfg.defined = make(map[string]bool, len(md.Keys()))
	for _, k := range md.Keys() {
		cfg.defined[k.String()] = true
	}
	cfg.sourcePath = abs
	return cfg, nil
}

// FindInDir looks for DefaultFileName inside dir.
//
// Returns the absolute path if found; ("", nil) when not present.
func FindInDir(dir string) (string, error) {
	p := filepath.Join(dir, DefaultFileName)
	info, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("stat %s: %w", p, err)
	}
	if info.IsDir() {
		return "", nil
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", p, err)
	}
	return abs, nil
}

// LoadFromCwd locates and loads the default config file from the current
// working directory.
//
// Returns (nil, nil) when no config file is found so the caller can
// distinguish "missing" from a parse error.
func LoadFromCwd() (*FileConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	p, err := FindInDir(cwd)
	if err != nil {
		return nil, err
	}
	if p == "" {
		return nil, nil
	}
	return Load(p)
}
