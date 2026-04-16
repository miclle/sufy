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
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/filesystem"
	"github.com/sufy-dev/sufy/sandbox/internal/envdapi/filesystem/filesystemconnect"
)

// FileType denotes a file-system entry's type.
type FileType string

const (
	// FileTypeFile is a regular file.
	FileTypeFile FileType = "file"
	// FileTypeDirectory is a directory.
	FileTypeDirectory FileType = "dir"
	// FileTypeUnknown is used when the server returns an entry type this SDK
	// does not yet know about.
	FileTypeUnknown FileType = "unknown"
)

// EntryInfo describes a file or directory.
type EntryInfo struct {
	Name          string
	Type          FileType
	Path          string
	Size          int64
	Mode          uint32
	Permissions   string
	Owner         string
	Group         string
	ModifiedTime  time.Time
	SymlinkTarget *string
}

func entryInfoFromProto(e *filesystem.EntryInfo) *EntryInfo {
	if e == nil {
		return nil
	}
	info := &EntryInfo{
		Name:        e.Name,
		Path:        e.Path,
		Size:        e.Size,
		Mode:        e.Mode,
		Permissions: e.Permissions,
		Owner:       e.Owner,
		Group:       e.Group,
	}
	switch e.Type {
	case filesystem.FileType_FILE_TYPE_FILE:
		info.Type = FileTypeFile
	case filesystem.FileType_FILE_TYPE_DIRECTORY:
		info.Type = FileTypeDirectory
	default:
		info.Type = FileTypeUnknown
	}
	if e.ModifiedTime != nil {
		info.ModifiedTime = e.ModifiedTime.AsTime()
	}
	if e.SymlinkTarget != nil {
		t := *e.SymlinkTarget
		info.SymlinkTarget = &t
	}
	return info
}

// EventType represents a filesystem change event.
type EventType string

const (
	// EventCreate is raised when a file or directory is created.
	EventCreate EventType = "create"
	// EventWrite is raised when a file is written to.
	EventWrite EventType = "write"
	// EventRemove is raised when a file or directory is deleted.
	EventRemove EventType = "remove"
	// EventRename is raised when a file or directory is renamed.
	EventRename EventType = "rename"
	// EventChmod is raised when permissions change.
	EventChmod EventType = "chmod"
)

// FilesystemEvent is a single change event observed by WatchDir.
type FilesystemEvent struct {
	Name string
	Type EventType
}

func filesystemEventFromProto(e *filesystem.FilesystemEvent) FilesystemEvent {
	ev := FilesystemEvent{Name: e.Name}
	switch e.Type {
	case filesystem.EventType_EVENT_TYPE_CREATE:
		ev.Type = EventCreate
	case filesystem.EventType_EVENT_TYPE_WRITE:
		ev.Type = EventWrite
	case filesystem.EventType_EVENT_TYPE_REMOVE:
		ev.Type = EventRemove
	case filesystem.EventType_EVENT_TYPE_RENAME:
		ev.Type = EventRename
	case filesystem.EventType_EVENT_TYPE_CHMOD:
		ev.Type = EventChmod
	}
	return ev
}

// FilesystemOption configures a filesystem operation.
type FilesystemOption func(*filesystemOpts)

type filesystemOpts struct {
	user string
}

// WithUser sets the OS user used for the filesystem operation.
func WithUser(user string) FilesystemOption {
	return func(o *filesystemOpts) { o.user = user }
}

func applyFilesystemOpts(opts []FilesystemOption) *filesystemOpts {
	o := &filesystemOpts{user: DefaultUser}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// ListOption configures a directory listing.
type ListOption func(*listOpts)

type listOpts struct {
	filesystemOpts
	depth uint32
}

// WithDepth sets the recursion depth for listings. Default is 1.
func WithDepth(depth uint32) ListOption {
	return func(o *listOpts) { o.depth = depth }
}

// WithListUser sets the OS user for directory listings.
func WithListUser(user string) ListOption {
	return func(o *listOpts) { o.user = user }
}

func applyListOpts(opts []ListOption) *listOpts {
	o := &listOpts{
		filesystemOpts: filesystemOpts{user: DefaultUser},
		depth:          1,
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WatchOption configures a WatchDir call.
type WatchOption func(*watchOpts)

type watchOpts struct {
	filesystemOpts
	recursive bool
}

// WithRecursive toggles recursive subdirectory watching.
func WithRecursive(recursive bool) WatchOption {
	return func(o *watchOpts) { o.recursive = recursive }
}

// WithWatchUser sets the OS user for the watch stream.
func WithWatchUser(user string) WatchOption {
	return func(o *watchOpts) { o.user = user }
}

func applyWatchOpts(opts []WatchOption) *watchOpts {
	o := &watchOpts{
		filesystemOpts: filesystemOpts{user: DefaultUser},
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WatchHandle streams filesystem events from WatchDir.
type WatchHandle struct {
	events chan FilesystemEvent
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

// Events returns the event channel. The channel is closed when the watch is
// stopped or when the stream ends.
func (w *WatchHandle) Events() <-chan FilesystemEvent {
	return w.events
}

// Err returns the first non-nil error observed by the stream goroutine. Only
// meaningful after Events() is closed. Stopping a watch via Stop() yields nil.
func (w *WatchHandle) Err() error {
	return w.err
}

// Stop cancels the watch and waits for the stream goroutine to exit.
func (w *WatchHandle) Stop() {
	w.cancel()
	<-w.done
}

// Filesystem exposes sandbox filesystem operations.
type Filesystem struct {
	sandbox *Sandbox
	rpc     filesystemconnect.FilesystemClient
}

// newFilesystem constructs a Filesystem sub-module.
func newFilesystem(s *Sandbox) *Filesystem {
	rpc := filesystemconnect.NewFilesystemClient(
		s.client.config.HTTPClient,
		s.envdURL(),
	)
	return &Filesystem{sandbox: s, rpc: rpc}
}

// checkHTTPResponse returns an APIError for any non-200 status code.
func checkHTTPResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return newAPIError(resp, body)
}

// Read downloads a file's entire contents.
func (fs *Filesystem) Read(ctx context.Context, path string, opts ...FilesystemOption) ([]byte, error) {
	resp, err := fs.doRead(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ReadText is Read but decodes the bytes as a UTF-8 string.
func (fs *Filesystem) ReadText(ctx context.Context, path string, opts ...FilesystemOption) (string, error) {
	data, err := fs.Read(ctx, path, opts...)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadStream returns the file's body as an io.ReadCloser. The caller must close
// the returned reader.
func (fs *Filesystem) ReadStream(ctx context.Context, path string, opts ...FilesystemOption) (io.ReadCloser, error) {
	resp, err := fs.doRead(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// doRead performs the GET /files request. The caller must close resp.Body.
func (fs *Filesystem) doRead(ctx context.Context, path string, opts ...FilesystemOption) (*http.Response, error) {
	o := applyFilesystemOpts(opts)
	downloadURL := fs.sandbox.DownloadURL(path, WithFileUser(o.user))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	setReqidHeader(ctx, req)

	httpClient := fs.sandbox.client.config.HTTPClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	if err := checkHTTPResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

// Write uploads the given data to path, overwriting if it exists. Parent
// directories are created as needed.
func (fs *Filesystem) Write(ctx context.Context, path string, data []byte, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	uploadURL := fs.sandbox.UploadURL(path, WithFileUser(o.user))

	pr, pw := io.Pipe()
	writer := newMultipartWriter(pw)

	go func() {
		if err := writer.writeFile("file", path, data); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := writer.close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.contentType())
	setReqidHeader(ctx, req)

	httpClient := fs.sandbox.client.config.HTTPClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPResponse(resp); err != nil {
		return nil, err
	}

	return fs.GetInfo(ctx, path, opts...)
}

// WriteEntry is a single entry for batch uploads.
type WriteEntry struct {
	Path string
	Data []byte
}

// WriteFiles uploads multiple files in one multipart POST. Paths are conveyed
// through each part's filename so the server can place them in one trip.
//
// After a successful upload, each file's metadata is refreshed via GetInfo,
// which makes this an N+1 operation. Prefer it for small batches.
func (fs *Filesystem) WriteFiles(ctx context.Context, files []WriteEntry, opts ...FilesystemOption) ([]*EntryInfo, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Fall back to the single-file path for one-file batches.
	if len(files) == 1 {
		info, err := fs.Write(ctx, files[0].Path, files[0].Data, opts...)
		if err != nil {
			return nil, err
		}
		return []*EntryInfo{info}, nil
	}

	o := applyFilesystemOpts(opts)
	uploadURL := fs.sandbox.batchUploadURL(o.user)

	pr, pw := io.Pipe()
	writer := newMultipartWriter(pw)

	go func() {
		for _, f := range files {
			if err := writer.writeFileFullPath("file", f.Path, f.Data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if err := writer.close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.contentType())
	setReqidHeader(ctx, req)

	httpClient := fs.sandbox.client.config.HTTPClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload files: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPResponse(resp); err != nil {
		return nil, err
	}

	infos := make([]*EntryInfo, 0, len(files))
	for _, f := range files {
		info, err := fs.GetInfo(ctx, f.Path, opts...)
		if err != nil {
			return nil, fmt.Errorf("get info for %s: %w", f.Path, err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// List returns entries in the given directory.
func (fs *Filesystem) List(ctx context.Context, path string, opts ...ListOption) ([]EntryInfo, error) {
	o := applyListOpts(opts)
	req := connect.NewRequest(&filesystem.ListDirRequest{
		Path:  path,
		Depth: o.depth,
	})
	fs.sandbox.setEnvdAuth(req, o.user)

	resp, err := fs.rpc.ListDir(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list dir: %w", err)
	}

	entries := make([]EntryInfo, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		info := entryInfoFromProto(e)
		if info == nil {
			continue
		}
		entries = append(entries, *info)
	}
	return entries, nil
}

// Exists reports whether the path exists.
func (fs *Filesystem) Exists(ctx context.Context, path string, opts ...FilesystemOption) (bool, error) {
	_, err := fs.GetInfo(ctx, path, opts...)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetInfo returns metadata for the given path.
func (fs *Filesystem) GetInfo(ctx context.Context, path string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.StatRequest{Path: path})
	fs.sandbox.setEnvdAuth(req, o.user)

	resp, err := fs.rpc.Stat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// MakeDir creates a directory, including any missing parent directories.
func (fs *Filesystem) MakeDir(ctx context.Context, path string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.MakeDirRequest{Path: path})
	fs.sandbox.setEnvdAuth(req, o.user)

	resp, err := fs.rpc.MakeDir(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// Remove deletes a file or directory.
func (fs *Filesystem) Remove(ctx context.Context, path string, opts ...FilesystemOption) error {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.RemoveRequest{Path: path})
	fs.sandbox.setEnvdAuth(req, o.user)

	_, err := fs.rpc.Remove(ctx, req)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

// Rename renames or moves a file or directory.
func (fs *Filesystem) Rename(ctx context.Context, oldPath, newPath string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.MoveRequest{
		Source:      oldPath,
		Destination: newPath,
	})
	fs.sandbox.setEnvdAuth(req, o.user)

	resp, err := fs.rpc.Move(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// WatchDir watches a directory for changes. The returned WatchHandle exposes
// event delivery and cancellation.
func (fs *Filesystem) WatchDir(ctx context.Context, path string, opts ...WatchOption) (*WatchHandle, error) {
	o := applyWatchOpts(opts)

	watchCtx, cancel := context.WithCancel(ctx)
	req := connect.NewRequest(&filesystem.WatchDirRequest{
		Path:      path,
		Recursive: o.recursive,
	})
	fs.sandbox.setEnvdAuth(req, o.user)

	stream, err := fs.rpc.WatchDir(watchCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("watch dir: %w", err)
	}

	w := &WatchHandle{
		events: make(chan FilesystemEvent, 64),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(w.done)
		defer close(w.events)
		for stream.Receive() {
			msg := stream.Msg()
			if fsEvent := msg.GetFilesystem(); fsEvent != nil {
				ev := filesystemEventFromProto(fsEvent)
				select {
				case w.events <- ev:
				case <-watchCtx.Done():
					return
				}
			}
		}
		if err := stream.Err(); err != nil && watchCtx.Err() == nil {
			w.err = err
		}
	}()

	return w, nil
}
