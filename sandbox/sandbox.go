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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	connect "connectrpc.com/connect"

	"github.com/sufy-dev/sufy/sandbox/internal/apis"
	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/process/processconnect"
	"github.com/sufy-dev/sufy/sandbox/internal/reqid"
)

// envdPort is the default TCP port that an envd agent listens on.
const envdPort = 49983

// DefaultUser is the default OS user used for command execution and file
// operations inside the sandbox.
const DefaultUser = "user"

// Sandbox represents a running sandbox instance. It holds a back-reference to
// the client for lifecycle calls and envd-agent communication.
type Sandbox struct {
	sandboxID          string
	templateID         string
	clientID           string
	alias              *string
	domain             *string
	trafficAccessToken *string

	// envdAccessToken is the credential used by envd RPCs. Access is guarded by
	// envdTokenMu for concurrent read/write safety.
	envdTokenMu     sync.RWMutex
	envdAccessToken *string
	envdTokenLoaded bool

	client *Client

	// Shared ProcessClient (used by both Commands and Pty).
	processRPCOnce sync.Once
	processRPC     processconnect.ProcessClient

	// envd sub-modules are initialized lazily.
	filesOnce sync.Once
	files     *Filesystem

	commandsOnce sync.Once
	commands     *Commands

	ptyOnce sync.Once
	pty     *Pty

	gitOnce sync.Once
	git     *Git
}

// newSandbox builds a Sandbox from an API payload.
func newSandbox(c *Client, s *apis.Sandbox) *Sandbox {
	sb := &Sandbox{
		sandboxID:          s.SandboxID,
		templateID:         s.TemplateID,
		clientID:           s.ClientID,
		alias:              s.Alias,
		domain:             s.Domain,
		trafficAccessToken: s.TrafficAccessToken,
		client:             c,
	}
	if s.EnvdAccessToken != nil {
		sb.envdAccessToken = s.EnvdAccessToken
		sb.envdTokenLoaded = true
	}
	return sb
}

// ID returns the sandbox identifier.
func (s *Sandbox) ID() string { return s.sandboxID }

// TemplateID returns the template the sandbox was created from.
func (s *Sandbox) TemplateID() string { return s.templateID }

// Alias returns the sandbox alias, if one was assigned.
func (s *Sandbox) Alias() *string { return s.alias }

// Domain returns the sandbox domain used to reach it externally.
func (s *Sandbox) Domain() *string { return s.domain }

// processClient returns the cached ProcessClient. Commands and Pty share a
// single underlying RPC client.
func (s *Sandbox) processClient() processconnect.ProcessClient {
	s.processRPCOnce.Do(func() {
		s.processRPC = processconnect.NewProcessClient(
			s.client.config.HTTPClient,
			s.envdURL(),
			connect.WithInterceptors(keepaliveInterceptor{}),
		)
	})
	return s.processRPC
}

// ---------------------------------------------------------------------------
// Client-level sandbox operations.
// ---------------------------------------------------------------------------

// Create spawns a new sandbox from the given template.
func (c *Client) Create(ctx context.Context, params CreateParams) (*Sandbox, error) {
	body, err := params.toAPI()
	if err != nil {
		return nil, err
	}
	resp, err := c.api.CreateSandboxWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	sb := newSandbox(c, resp.JSON201)
	// Create may not echo back envdAccessToken; fetch it via GetSandbox so
	// envd RPCs (PTY, commands, filesystem) authenticate correctly.
	if !sb.envdTokenLoaded {
		if err := sb.refreshEnvdToken(ctx); err != nil {
			return nil, fmt.Errorf("create sandbox %s: %w", sb.sandboxID, err)
		}
	}
	return sb, nil
}

// Connect attaches to an existing sandbox and optionally resumes a paused one.
func (c *Client) Connect(ctx context.Context, sandboxID string, params ConnectParams) (*Sandbox, error) {
	resp, err := c.api.ConnectSandboxWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	var sb *Sandbox
	if resp.JSON200 != nil {
		sb = newSandbox(c, resp.JSON200)
	} else if resp.JSON201 != nil {
		sb = newSandbox(c, resp.JSON201)
	} else {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	if !sb.envdTokenLoaded {
		if err := sb.refreshEnvdToken(ctx); err != nil {
			return nil, fmt.Errorf("connect sandbox %s: %w", sandboxID, err)
		}
	}
	return sb, nil
}

// List returns a page of sandboxes matching the given filters.
func (c *Client) List(ctx context.Context, __xgo_optional_params *ListParams) ([]ListedSandbox, error) {
	resp, err := c.api.ListSandboxesV2WithResponse(ctx, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return listedSandboxesFromAPI(*resp.JSON200), nil
}

// CreateAndWait creates a sandbox and blocks until it reaches the running state.
func (c *Client) CreateAndWait(ctx context.Context, params CreateParams, opts ...PollOption) (*Sandbox, *SandboxInfo, error) {
	sb, err := c.Create(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("create sandbox: %w", err)
	}
	info, err := sb.WaitForReady(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}
	return sb, info, nil
}

// GetSandboxesMetrics returns metrics for the given sandbox IDs.
func (c *Client) GetSandboxesMetrics(ctx context.Context, __xgo_optional_params *GetSandboxesMetricsParams) (*SandboxesWithMetrics, error) {
	resp, err := c.api.GetSandboxesMetricsWithResponse(ctx, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxesWithMetricsFromAPI(resp.JSON200), nil
}

// ---------------------------------------------------------------------------
// Package-level shortcuts (xgo-friendly).
// ---------------------------------------------------------------------------

// Create spawns a new sandbox using a client derived from environment defaults.
func Create(ctx context.Context, params CreateParams) (*Sandbox, error) {
	return New(nil).Create(ctx, params)
}

// List returns a page of sandboxes using a client derived from environment defaults.
func List(ctx context.Context, __xgo_optional_params *ListParams) ([]ListedSandbox, error) {
	return New(nil).List(ctx, __xgo_optional_params)
}

// Connect attaches to an existing sandbox using a client derived from environment defaults.
func Connect(ctx context.Context, sandboxID string, params ConnectParams) (*Sandbox, error) {
	return New(nil).Connect(ctx, sandboxID, params)
}

// ---------------------------------------------------------------------------
// Sandbox-level lifecycle operations.
// ---------------------------------------------------------------------------

// Kill terminates the sandbox.
func (s *Sandbox) Kill(ctx context.Context) error {
	resp, err := s.client.api.DeleteSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// SetTimeout updates the sandbox's remaining lifetime. The sandbox will expire
// after the given duration from now. Timeout must be at least one second.
func (s *Sandbox) SetTimeout(ctx context.Context, timeout time.Duration) error {
	if timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", timeout)
	}
	secs := timeout.Seconds()
	if secs > float64(math.MaxInt32) {
		return fmt.Errorf("timeout %v exceeds maximum allowed value", timeout)
	}
	timeoutSec := int32(secs)
	resp, err := s.client.api.UpdateSandboxTimeoutWithResponse(ctx, s.sandboxID, apis.UpdateSandboxTimeoutJSONRequestBody{
		Timeout: timeoutSec,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// refreshEnvdToken fetches envdAccessToken via GetSandbox and stores it on the
// Sandbox. Called internally when Create/Connect did not include the token.
func (s *Sandbox) refreshEnvdToken(ctx context.Context) error {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox %s for envd token: %w", s.sandboxID, err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("get sandbox %s for envd token: %w", s.sandboxID, newAPIError(resp.HTTPResponse, resp.Body))
	}
	s.envdTokenMu.Lock()
	s.envdAccessToken = resp.JSON200.EnvdAccessToken
	s.envdTokenLoaded = true
	s.envdTokenMu.Unlock()
	return nil
}

// GetInfo returns the sandbox's full details.
func (s *Sandbox) GetInfo(ctx context.Context) (*SandboxInfo, error) {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxInfoFromAPI(resp.JSON200), nil
}

// IsRunning probes the sandbox's envd /health endpoint to confirm the agent is
// reachable. Unlike GetInfo, which queries the control-plane status, this
// verifies the in-sandbox agent is up. Returns false when the sandbox is paused
// or otherwise unreachable without returning an error.
func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.envdURL()+"/health", nil)
	if err != nil {
		return false, err
	}
	setReqidHeader(ctx, req)
	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	}
	if resp.StatusCode == http.StatusBadGateway {
		return false, nil
	}
	return false, newAPIError(resp, nil)
}

// GetMetrics returns the sandbox's resource metrics.
func (s *Sandbox) GetMetrics(ctx context.Context, __xgo_optional_params *GetMetricsParams) ([]SandboxMetric, error) {
	resp, err := s.client.api.GetSandboxMetricsWithResponse(ctx, s.sandboxID, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxMetricsFromAPI(*resp.JSON200), nil
}

// GetLogs returns the sandbox's log output.
func (s *Sandbox) GetLogs(ctx context.Context, __xgo_optional_params *GetLogsParams) (*SandboxLogs, error) {
	resp, err := s.client.api.GetSandboxLogsWithResponse(ctx, s.sandboxID, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxLogsFromAPI(resp.JSON200), nil
}

// Pause pauses the sandbox so it can be resumed later.
func (s *Sandbox) Pause(ctx context.Context) error {
	resp, err := s.client.api.PauseSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// Refresh extends the sandbox's lifetime.
func (s *Sandbox) Refresh(ctx context.Context, params RefreshParams) error {
	resp, err := s.client.api.RefreshSandboxWithResponse(ctx, s.sandboxID, params.toAPI())
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// WaitForReady polls GetInfo until the sandbox reaches the running state or
// the context is cancelled. Defaults to a 1-second polling interval.
func (s *Sandbox) WaitForReady(ctx context.Context, opts ...PollOption) (*SandboxInfo, error) {
	o := defaultPollOpts(time.Second)
	for _, fn := range opts {
		fn(o)
	}

	return pollLoop(ctx, o, func() (bool, *SandboxInfo, error) {
		info, err := s.GetInfo(ctx)
		if err != nil {
			return false, nil, fmt.Errorf("get sandbox %s: %w", s.sandboxID, err)
		}
		if info.State == StateRunning {
			return true, info, nil
		}
		return false, nil, nil
	})
}

// ---------------------------------------------------------------------------
// envd helpers
// ---------------------------------------------------------------------------

// Files returns the filesystem sub-module, lazily initialized.
func (s *Sandbox) Files() *Filesystem {
	s.filesOnce.Do(func() {
		s.files = newFilesystem(s)
	})
	return s.files
}

// Commands returns the commands sub-module, lazily initialized.
func (s *Sandbox) Commands() *Commands {
	s.commandsOnce.Do(func() {
		s.commands = newCommands(s, s.processClient())
	})
	return s.commands
}

// Pty returns the PTY sub-module, lazily initialized.
func (s *Sandbox) Pty() *Pty {
	s.ptyOnce.Do(func() {
		s.pty = newPty(s, s.processClient())
	})
	return s.pty
}

// Git returns the git operations sub-module, lazily initialized. The sandbox
// must have the git binary preinstalled; only HTTPS + username/password
// (token) authentication is supported.
func (s *Sandbox) Git() *Git {
	s.gitOnce.Do(func() {
		s.git = newGit(s.Commands())
	})
	return s.git
}

// GetHost returns the external hostname through which the given port on this
// sandbox can be reached. Format: "{port}-{sandboxID}.{domain}".
func (s *Sandbox) GetHost(port int) string {
	if s.domain == nil || *s.domain == "" {
		return ""
	}
	return fmt.Sprintf("%d-%s.%s", port, s.sandboxID, *s.domain)
}

// envdURL returns the base URL of the envd agent for this sandbox.
func (s *Sandbox) envdURL() string {
	return fmt.Sprintf("https://%s", s.GetHost(envdPort))
}

// envdBasicAuth builds the Basic auth value used to convey the OS user to envd.
// The password component is empty; envd uses a separate X-Access-Token header
// for access control.
func envdBasicAuth(user string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"))
}

// setEnvdAuth attaches the two envd auth headers to a ConnectRPC request:
//
//   - Authorization: Basic base64(user:) — conveys the OS user identity.
//   - X-Access-Token: <token> — envd's access token (when present).
func (s *Sandbox) setEnvdAuth(req interface{ Header() http.Header }, user string) {
	req.Header().Set("Authorization", envdBasicAuth(user))
	s.envdTokenMu.RLock()
	tok := s.envdAccessToken
	s.envdTokenMu.RUnlock()
	if tok != nil && *tok != "" {
		req.Header().Set("X-Access-Token", *tok)
	}
}

// keepalivePingIntervalSec is the keep-alive ping interval (seconds) used on
// streaming connections. It matches the JS SDK value so intermediate proxies
// do not close idle streams.
const keepalivePingIntervalSec = "50"

// keepalivePingHeader is the HTTP header used to advertise the keep-alive interval.
const keepalivePingHeader = "Keepalive-Ping-Interval"

// keepaliveInterceptor is a ConnectRPC interceptor that injects the
// Keepalive-Ping-Interval header on all streaming requests.
type keepaliveInterceptor struct{}

func (keepaliveInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (keepaliveInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(keepalivePingHeader, keepalivePingIntervalSec)
		return conn
	}
}

func (keepaliveInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// setReqidHeader injects the X-Reqid header from the context's request ID.
// Used for direct HTTP calls that bypass the oapi-codegen client (e.g. envd
// filesystem upload/download).
func setReqidHeader(ctx context.Context, req *http.Request) {
	if id, ok := reqid.ReqidFromContext(ctx); ok {
		req.Header.Set("X-Reqid", id)
	}
}

// ---------------------------------------------------------------------------
// envd file URL helpers
// ---------------------------------------------------------------------------

// FileURLOption configures DownloadURL and UploadURL.
type FileURLOption func(*fileURLOpts)

type fileURLOpts struct {
	user                string
	signatureExpiration int
}

// WithFileUser sets the OS user for file operations.
func WithFileUser(user string) FileURLOption {
	return func(o *fileURLOpts) { o.user = user }
}

// WithSignatureExpiration sets the signature validity window in seconds.
func WithSignatureExpiration(seconds int) FileURLOption {
	return func(o *fileURLOpts) { o.signatureExpiration = seconds }
}

// fileSignature computes a file-operation signature.
//
// Algorithm: "v1_" + SHA256(path + ":" + operation + ":" + username + ":" + accessToken + ":" + expiration)
//
// The signature algorithm is defined by the server; the client must keep it in
// lock-step. The current format is not HMAC-based and uses ':' as a separator,
// meaning it has theoretical forgery and collision risks if the access token
// leaks or if usernames/paths embed ':'. Hardening is expected to be driven
// from the server side.
func fileSignature(path, operation, username, accessToken string, expiration int) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%d", path, operation, username, accessToken, expiration)
	hash := sha256.Sum256([]byte(raw))
	return "v1_" + fmt.Sprintf("%x", hash)
}

// DownloadURL returns a signed URL for downloading a file from the sandbox.
func (s *Sandbox) DownloadURL(path string, opts ...FileURLOption) string {
	return s.fileURL(path, "read", opts...)
}

// UploadURL returns a signed URL for uploading a file to the sandbox via
// multipart/form-data POST.
func (s *Sandbox) UploadURL(path string, opts ...FileURLOption) string {
	return s.fileURL(path, "write", opts...)
}

// fileURL builds a signed envd file-operation URL.
func (s *Sandbox) fileURL(path, operation string, opts ...FileURLOption) string {
	o := &fileURLOpts{user: DefaultUser}
	for _, fn := range opts {
		fn(o)
	}

	q := url.Values{}
	q.Set("path", path)
	q.Set("username", o.user)

	s.envdTokenMu.RLock()
	tok := s.envdAccessToken
	s.envdTokenMu.RUnlock()
	if tok != nil && *tok != "" {
		exp := o.signatureExpiration
		if exp == 0 {
			exp = 300
		}
		sig := fileSignature(path, operation, o.user, *tok, exp)
		q.Set("signature", sig)
		q.Set("signature_expiration", strconv.Itoa(exp))
	}

	return s.envdURL() + "/files?" + q.Encode()
}

// batchUploadURL is the upload URL for batch multipart uploads. Paths are
// carried by each part's filename rather than as a query parameter.
func (s *Sandbox) batchUploadURL(user string) string {
	q := url.Values{}
	q.Set("username", user)
	return s.envdURL() + "/files?" + q.Encode()
}
