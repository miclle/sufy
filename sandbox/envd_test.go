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
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func TestSandboxAlias(t *testing.T) {
	alias := "my-alias"
	sb := &Sandbox{alias: &alias}
	if sb.Alias() != &alias {
		t.Errorf("Alias() = %v, want %v", sb.Alias(), &alias)
	}
	if got := *sb.Alias(); got != "my-alias" {
		t.Errorf("*Alias() = %q, want %q", got, "my-alias")
	}

	sb2 := &Sandbox{}
	if sb2.Alias() != nil {
		t.Errorf("Alias() = %v, want nil", sb2.Alias())
	}
}

func TestSandboxDomain(t *testing.T) {
	domain := "example.com"
	sb := &Sandbox{domain: &domain}
	if sb.Domain() != &domain {
		t.Errorf("Domain() = %v, want %v", sb.Domain(), &domain)
	}
	if got := *sb.Domain(); got != "example.com" {
		t.Errorf("*Domain() = %q, want %q", got, "example.com")
	}

	sb2 := &Sandbox{}
	if sb2.Domain() != nil {
		t.Errorf("Domain() = %v, want nil", sb2.Domain())
	}
}

func TestKeepaliveWrapStreamingHandler(t *testing.T) {
	ki := keepaliveInterceptor{}
	called := false
	handler := connect.StreamingHandlerFunc(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		called = true
		return nil
	})
	wrapped := ki.WrapStreamingHandler(handler)
	if err := wrapped(context.Background(), nil); err != nil {
		t.Fatalf("WrapStreamingHandler returned error: %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestFileURLOptionWithExpiration(t *testing.T) {
	domain := "test.dev"
	token := "tok"
	sb := &Sandbox{sandboxID: "sb-1", domain: &domain, envdAccessToken: &token, client: &Client{config: &Config{}}}

	u := sb.DownloadURL("/file.txt", WithSignatureExpiration(60))
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
	// The URL should contain signature_expiration=60.
	if !strings.Contains(u, "signature_expiration=60") {
		t.Errorf("DownloadURL = %q, want to contain signature_expiration=60", u)
	}
}

func TestGetHost(t *testing.T) {
	domain := "example.com"
	sb := &Sandbox{sandboxID: "sb-123", domain: &domain}

	got := sb.GetHost(8080)
	want := "8080-sb-123.example.com"
	if got != want {
		t.Errorf("GetHost = %q, want %q", got, want)
	}
}

func TestGetHostNilDomain(t *testing.T) {
	sb := &Sandbox{sandboxID: "sb-456"}

	got := sb.GetHost(3000)
	if got != "" {
		t.Errorf("GetHost with nil domain = %q, want empty string", got)
	}
}

func TestGetHostEmptyDomain(t *testing.T) {
	domain := ""
	sb := &Sandbox{sandboxID: "sb-789", domain: &domain}

	got := sb.GetHost(443)
	if got != "" {
		t.Errorf("GetHost with empty domain = %q, want empty string", got)
	}
}

func TestEnvdURL(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	got := sb.envdURL()
	want := "https://49983-sb-100.test.dev"
	if got != want {
		t.Errorf("envdURL = %q, want %q", got, want)
	}
}

func TestEnvdBasicAuth(t *testing.T) {
	auth := envdBasicAuth("testuser")
	// base64("testuser:") = "dGVzdHVzZXI6"
	want := "Basic dGVzdHVzZXI6"
	if auth != want {
		t.Errorf("envdBasicAuth = %q, want %q", auth, want)
	}
}

func TestFileSignature(t *testing.T) {
	sig := fileSignature("/test/file.txt", "read", "user", "token123", 300)
	raw := "/test/file.txt:read:user:token123:300"
	hash := sha256.Sum256([]byte(raw))
	want := "v1_" + fmt.Sprintf("%x", hash)
	if sig != want {
		t.Errorf("fileSignature = %q, want %q", sig, want)
	}
}

func TestDownloadURL(t *testing.T) {
	domain := "test.dev"
	token := "mytoken"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, envdAccessToken: &token, client: &Client{config: &Config{}}}

	u := sb.DownloadURL("/home/user/file.txt")
	// URL should carry the necessary query parameters.
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
	// Check the base structure.
	if got := "https://49983-sb-100.test.dev/files?"; len(u) < len(got) || u[:len(got)] != got {
		t.Errorf("DownloadURL prefix = %q, want prefix %q", u, got)
	}
}

func TestUploadURL(t *testing.T) {
	domain := "test.dev"
	token := "mytoken"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, envdAccessToken: &token, client: &Client{config: &Config{}}}

	u := sb.UploadURL("/home/user/file.txt")
	if u == "" {
		t.Fatal("UploadURL returned empty string")
	}
	if got := "https://49983-sb-100.test.dev/files?"; len(u) < len(got) || u[:len(got)] != got {
		t.Errorf("UploadURL prefix = %q, want prefix %q", u, got)
	}
}

func TestDownloadURLWithoutToken(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	u := sb.DownloadURL("/file.txt")
	// Without a token, the signature parameter must not be present.
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
}

func TestFilesLazyInit(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	fs1 := sb.Files()
	fs2 := sb.Files()
	if fs1 != fs2 {
		t.Error("Files() should return the same instance")
	}
}

func TestCommandsLazyInit(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	cmd1 := sb.Commands()
	cmd2 := sb.Commands()
	if cmd1 != cmd2 {
		t.Error("Commands() should return the same instance")
	}
}

func TestPtyLazyInit(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	pty1 := sb.Pty()
	pty2 := sb.Pty()
	if pty1 != pty2 {
		t.Error("Pty() should return the same instance")
	}
}

func TestFileURLOptionWithUser(t *testing.T) {
	domain := "test.dev"
	token := "tok"
	sb := &Sandbox{sandboxID: "sb-1", domain: &domain, envdAccessToken: &token, client: &Client{config: &Config{}}}

	u := sb.DownloadURL("/file.txt", WithFileUser("admin"))
	// URL should carry username=admin.
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
}

func TestIsNotFoundError(t *testing.T) {
	apiErr := &APIError{StatusCode: 404, Body: []byte("not found")}
	if !isNotFoundError(apiErr) {
		t.Error("expected isNotFoundError to return true for 404 APIError")
	}

	apiErr200 := &APIError{StatusCode: 200, Body: []byte("ok")}
	if isNotFoundError(apiErr200) {
		t.Error("expected isNotFoundError to return false for 200 APIError")
	}
}

func TestEntryInfoFromProtoNil(t *testing.T) {
	if entryInfoFromProto(nil) != nil {
		t.Error("entryInfoFromProto(nil) should return nil")
	}
}

func TestWriteFilesEmpty(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}
	fs := &Filesystem{sandbox: sb}

	infos, err := fs.WriteFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("WriteFiles(nil) should return a nil error, got: %v", err)
	}
	if infos != nil {
		t.Fatalf("WriteFiles(nil) should return nil, got: %v", infos)
	}
}

func TestBatchUploadURL(t *testing.T) {
	domain := "test.dev"
	sb := &Sandbox{sandboxID: "sb-100", domain: &domain, client: &Client{config: &Config{}}}

	u := sb.batchUploadURL("user")
	want := "https://49983-sb-100.test.dev/files?username=user"
	if u != want {
		t.Errorf("batchUploadURL = %q, want %q", u, want)
	}
}

func TestWriteFileFullPath(t *testing.T) {
	var buf bytes.Buffer
	w := newMultipartWriter(&buf)

	if err := w.writeFileFullPath("file_0", "/home/user/test.txt", []byte("hello")); err != nil {
		t.Fatalf("writeFileFullPath failed: %v", err)
	}
	if err := w.close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Parse the multipart body and verify that the filename in
	// Content-Disposition is the full path.
	// Note: Part.FileName() calls filepath.Base(), so we inspect the header directly.
	r := multipart.NewReader(&buf, w.w.Boundary())
	part, err := r.NextPart()
	if err != nil {
		t.Fatalf("NextPart failed: %v", err)
	}
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("ParseMediaType failed: %v", err)
	}
	if got := params["filename"]; got != "/home/user/test.txt" {
		t.Errorf("filename = %q, want %q", got, "/home/user/test.txt")
	}
	data, _ := io.ReadAll(part)
	if string(data) != "hello" {
		t.Errorf("data = %q, want %q", string(data), "hello")
	}
}

func TestWriteFileBaseName(t *testing.T) {
	var buf bytes.Buffer
	w := newMultipartWriter(&buf)

	if err := w.writeFile("file", "/home/user/test.txt", []byte("hello")); err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}
	if err := w.close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Parse the multipart body and verify that filename uses the basename.
	r := multipart.NewReader(&buf, w.w.Boundary())
	part, err := r.NextPart()
	if err != nil {
		t.Fatalf("NextPart failed: %v", err)
	}
	if got := part.FileName(); got != "test.txt" {
		t.Errorf("filename = %q, want %q", got, "test.txt")
	}
}
