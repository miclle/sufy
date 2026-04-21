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
	"strings"
	"testing"
)

func TestConvert_Basic(t *testing.T) {
	content := "FROM alpine:3.19\nRUN echo hello\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BaseImage != "alpine:3.19" {
		t.Errorf("unexpected base image: %q", result.BaseImage)
	}
	// Should prepend USER root + WORKDIR /, append USER user + WORKDIR /home/user.
	if len(result.Steps) < 5 {
		t.Fatalf("expected at least 5 steps, got %d: %+v", len(result.Steps), result.Steps)
	}
	if result.Steps[0].Type != "USER" || (*result.Steps[0].Args)[0] != "root" {
		t.Errorf("expected first step USER root, got %+v", result.Steps[0])
	}
	if result.Steps[1].Type != "WORKDIR" || (*result.Steps[1].Args)[0] != "/" {
		t.Errorf("expected second step WORKDIR /, got %+v", result.Steps[1])
	}
	last := result.Steps[len(result.Steps)-1]
	if last.Type != "WORKDIR" || (*last.Args)[0] != "/home/user" {
		t.Errorf("expected last step WORKDIR /home/user, got %+v", last)
	}
}

func TestConvert_NoFROMError(t *testing.T) {
	_, err := Convert("RUN echo hi\n")
	if err == nil || !strings.Contains(err.Error(), "no FROM") {
		t.Errorf("expected no FROM error, got %v", err)
	}
}

func TestConvert_EmptyRUN(t *testing.T) {
	_, err := Convert("FROM alpine\nRUN\n")
	if err == nil || !strings.Contains(err.Error(), "empty RUN") {
		t.Errorf("expected empty RUN error, got %v", err)
	}
}

func TestConvert_EmptyWORKDIR(t *testing.T) {
	_, err := Convert("FROM alpine\nWORKDIR\n")
	if err == nil || !strings.Contains(err.Error(), "empty WORKDIR") {
		t.Errorf("expected empty WORKDIR error, got %v", err)
	}
}

func TestConvert_EmptyUSER(t *testing.T) {
	_, err := Convert("FROM alpine\nUSER\n")
	if err == nil || !strings.Contains(err.Error(), "empty USER") {
		t.Errorf("expected empty USER error, got %v", err)
	}
}

func TestConvert_UserAndWorkdirPreventDefaults(t *testing.T) {
	content := "FROM alpine\nUSER alice\nWORKDIR /srv\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var userCount, workdirCount int
	for _, s := range result.Steps {
		switch s.Type {
		case "USER":
			userCount++
		case "WORKDIR":
			workdirCount++
		}
	}
	// Expect: one prepended USER root, one explicit USER alice. No trailing USER user.
	if userCount != 2 {
		t.Errorf("expected 2 USER steps, got %d", userCount)
	}
	// Expect: one prepended WORKDIR /, one explicit WORKDIR /srv. No trailing default.
	if workdirCount != 2 {
		t.Errorf("expected 2 WORKDIR steps, got %d", workdirCount)
	}
}

func TestConvert_COPYWithChown(t *testing.T) {
	content := "FROM alpine\nCOPY --chown=alice:grp src dest\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var copyStep *struct {
		args []string
	}
	for _, s := range result.Steps {
		if s.Type == "COPY" {
			copyStep = &struct{ args []string }{args: *s.Args}
			break
		}
	}
	if copyStep == nil {
		t.Fatalf("expected COPY step")
	}
	// Args layout: [src, dest, user, ""]
	if copyStep.args[0] != "src" || copyStep.args[1] != "dest" || copyStep.args[2] != "alice" {
		t.Errorf("unexpected COPY args: %v", copyStep.args)
	}
}

func TestConvert_COPYMissingFields(t *testing.T) {
	_, err := Convert("FROM alpine\nCOPY only\n")
	if err == nil || !strings.Contains(err.Error(), "invalid COPY") {
		t.Errorf("expected invalid COPY error, got %v", err)
	}
}

func TestConvert_ENV(t *testing.T) {
	content := "FROM alpine\nENV K1=v1 K2=\"val 2\"\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var foundENV bool
	for _, s := range result.Steps {
		if s.Type == "ENV" {
			foundENV = true
			args := *s.Args
			if args[0] != "K1" || args[1] != "v1" || args[2] != "K2" || args[3] != "val 2" {
				t.Errorf("unexpected ENV args: %v", args)
			}
		}
	}
	if !foundENV {
		t.Errorf("expected ENV step")
	}
}

func TestConvert_ARGWithDefaultBecomesENV(t *testing.T) {
	content := "FROM alpine\nARG foo=bar\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found bool
	for _, s := range result.Steps {
		if s.Type == "ENV" && (*s.Args)[0] == "foo" && (*s.Args)[1] == "bar" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ARG foo=bar to become ENV step")
	}
}

func TestConvert_ARGWithoutDefaultIgnored(t *testing.T) {
	content := "FROM alpine\nARG foo\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range result.Steps {
		if s.Type == "ENV" {
			t.Errorf("ARG without default should not produce ENV step")
		}
	}
}

func TestConvert_CMDSetsStartAndReady(t *testing.T) {
	content := "FROM alpine\nCMD [\"/app\"]\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartCmd != "/app" {
		t.Errorf("unexpected StartCmd: %q", result.StartCmd)
	}
	if result.ReadyCmd != "sleep 20" {
		t.Errorf("unexpected ReadyCmd: %q", result.ReadyCmd)
	}
}

func TestConvert_ENTRYPOINTSetsStartCmd(t *testing.T) {
	content := "FROM alpine\nENTRYPOINT [\"/bin/app\"]\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartCmd == "" {
		t.Errorf("expected StartCmd to be set")
	}
}

func TestConvert_MultiStageWarns(t *testing.T) {
	content := "FROM alpine AS build\nFROM alpine AS runtime\n"
	result, err := Convert(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BaseImage != "alpine" {
		t.Errorf("expected last-stage base image alpine, got %q", result.BaseImage)
	}
	var found bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "multi-stage") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected multi-stage warning, got %v", result.Warnings)
	}
}

func TestExtractImage_WithAS(t *testing.T) {
	if got := extractImage("alpine:3.19 AS builder"); got != "alpine:3.19" {
		t.Errorf("got %q", got)
	}
}

func TestExtractImage_WithoutAS(t *testing.T) {
	if got := extractImage("debian:bookworm"); got != "debian:bookworm" {
		t.Errorf("got %q", got)
	}
}

func TestParseCopyArgs_ChownUserOnly(t *testing.T) {
	user, src, dest, err := parseCopyArgs("src dest", map[string]string{"chown": "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "alice" || src != "src" || dest != "dest" {
		t.Errorf("got user=%q src=%q dest=%q", user, src, dest)
	}
}

func TestParseCopyArgs_MultipleSources(t *testing.T) {
	_, src, dest, err := parseCopyArgs("a b c dest", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "a b c" || dest != "dest" {
		t.Errorf("got src=%q dest=%q", src, dest)
	}
}

func TestParseArgValues(t *testing.T) {
	kv, hasDefault := parseArgValues("foo=bar")
	if !hasDefault || kv[0] != "foo" || kv[1] != "bar" {
		t.Errorf("got %v hasDefault=%v", kv, hasDefault)
	}
	kv, hasDefault = parseArgValues("foo")
	if hasDefault || kv[0] != "foo" || kv[1] != "" {
		t.Errorf("got %v hasDefault=%v", kv, hasDefault)
	}
}
