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

package dockerfile

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ComputeFilesHash computes a SHA-256 hash of the files matching a COPY instruction.
// It matches e2b's calculateFilesHash algorithm:
//  1. hash.Update("COPY src dest")
//  2. For each file (sorted by POSIX relative path):
//     hash.Update(relative POSIX path)
//     hash.Update(file mode)
//     hash.Update(file size)
//     hash.Update(file contents)
func ComputeFilesHash(src, dest, contextPath string, ignorePatterns []string) (string, error) {
	files, err := collectFiles(src, contextPath, ignorePatterns)
	if err != nil {
		return "", fmt.Errorf("collect files for hash: %w", err)
	}

	h := sha256.New()
	fmt.Fprintf(h, "COPY %s %s", src, dest)

	for _, f := range files {
		h.Write([]byte(f.relPath))
		fmt.Fprintf(h, "%d", f.info.Mode())
		fmt.Fprintf(h, "%d", f.info.Size())

		file, err := os.Open(f.absPath)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", f.absPath, err)
		}
		_, err = io.Copy(h, file)
		file.Close()
		if err != nil {
			return "", fmt.Errorf("hash file %s: %w", f.absPath, err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// CollectAndUpload gathers files matching the src pattern, streams them as a
// gzip-compressed tar archive via HTTP PUT to uploadURL.
// Uses io.Pipe for zero-copy streaming (no full-archive buffering in memory).
func CollectAndUpload(ctx context.Context, uploadURL, src, contextPath string, ignorePatterns []string) error {
	files, err := collectFiles(src, contextPath, ignorePatterns)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		gw := gzip.NewWriter(pw)
		tw := tar.NewWriter(gw)

		var writeErr error
		for _, f := range files {
			header, err := tar.FileInfoHeader(f.info, "")
			if err != nil {
				writeErr = fmt.Errorf("create tar header for %s: %w", f.relPath, err)
				break
			}
			header.Name = f.relPath

			if err := tw.WriteHeader(header); err != nil {
				writeErr = fmt.Errorf("write tar header for %s: %w", f.relPath, err)
				break
			}

			file, err := os.Open(f.absPath)
			if err != nil {
				writeErr = fmt.Errorf("open file %s: %w", f.absPath, err)
				break
			}
			_, err = io.Copy(tw, file)
			file.Close()
			if err != nil {
				writeErr = fmt.Errorf("write file %s to tar: %w", f.relPath, err)
				break
			}
		}

		tw.Close()
		gw.Close()
		if writeErr != nil {
			pw.CloseWithError(writeErr)
		} else {
			pw.Close()
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, pr)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReadDockerignore reads .dockerignore patterns from the build context directory.
func ReadDockerignore(contextPath string) []string {
	path := filepath.Join(contextPath, ".dockerignore")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// fileEntry represents a collected file for hashing or uploading.
type fileEntry struct {
	absPath string
	relPath string // POSIX-style relative path
	info    fs.FileInfo
}

// collectFiles walks the build context directory, matching the source path or glob pattern,
// filtering by ignore rules. Results are sorted by relative path.
func collectFiles(src, contextPath string, ignorePatterns []string) ([]fileEntry, error) {
	contextAbs, err := filepath.Abs(contextPath)
	if err != nil {
		return nil, fmt.Errorf("resolve context path: %w", err)
	}

	srcAbs := src
	if !filepath.IsAbs(src) {
		srcAbs = filepath.Join(contextPath, src)
	}
	srcAbs, err = filepath.Abs(srcAbs)
	if err != nil {
		return nil, fmt.Errorf("resolve source path: %w", err)
	}

	// Path traversal check.
	if !isWithinContext(srcAbs, contextAbs) {
		return nil, fmt.Errorf("path %q escapes build context %q", src, contextPath)
	}

	info, err := os.Stat(srcAbs)
	if err != nil {
		return collectGlob(srcAbs, contextPath, ignorePatterns)
	}

	var files []fileEntry
	if info.IsDir() {
		err := filepath.Walk(srcAbs, func(path string, fi fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(contextPath, path)
			posixRel := filepath.ToSlash(rel)
			if isIgnored(posixRel, ignorePatterns) {
				return nil
			}
			files = append(files, fileEntry{
				absPath: path,
				relPath: posixRel,
				info:    fi,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		rel, _ := filepath.Rel(contextPath, srcAbs)
		posixRel := filepath.ToSlash(rel)
		if !isIgnored(posixRel, ignorePatterns) {
			files = append(files, fileEntry{
				absPath: srcAbs,
				relPath: posixRel,
				info:    info,
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})
	return files, nil
}

// collectGlob uses a glob pattern to match and collect files.
func collectGlob(pattern, contextPath string, ignorePatterns []string) ([]fileEntry, error) {
	contextAbs, _ := filepath.Abs(contextPath)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	var files []fileEntry
	for _, match := range matches {
		absMatch, _ := filepath.Abs(match)
		if !isWithinContext(absMatch, contextAbs) {
			continue
		}
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		rel, _ := filepath.Rel(contextPath, match)
		posixRel := filepath.ToSlash(rel)
		if isIgnored(posixRel, ignorePatterns) {
			continue
		}
		files = append(files, fileEntry{
			absPath: match,
			relPath: posixRel,
			info:    info,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})
	return files, nil
}

// isIgnored checks whether a path matches the ignore patterns.
// Uses last-match-wins semantics, matching Docker's .dockerignore spec.
func isIgnored(relPath string, patterns []string) bool {
	ignored := false
	for _, p := range patterns {
		negated := strings.HasPrefix(p, "!")
		if negated {
			p = p[1:]
		}

		matched, err := filepath.Match(p, relPath)
		if err != nil {
			continue
		}
		if !matched {
			matched, _ = filepath.Match(p, filepath.Base(relPath))
		}
		if matched {
			ignored = !negated
		}
	}
	return ignored
}

// isWithinContext checks whether a path is inside the build context directory.
func isWithinContext(path, contextAbs string) bool {
	return strings.HasPrefix(path+string(filepath.Separator), contextAbs+string(filepath.Separator)) ||
		path == contextAbs
}
