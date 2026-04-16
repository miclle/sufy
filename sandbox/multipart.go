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
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
)

// multipartFileWriter wraps a multipart.Writer for file uploads.
type multipartFileWriter struct {
	w *multipart.Writer
}

func newMultipartWriter(w io.Writer) *multipartFileWriter {
	return &multipartFileWriter{w: multipart.NewWriter(w)}
}

func (m *multipartFileWriter) contentType() string {
	return m.w.FormDataContentType()
}

// writeFile writes a single file part. The part's filename uses the basename of
// the supplied path, matching multipart.Writer.CreateFormFile semantics.
func (m *multipartFileWriter) writeFile(fieldName, fileName string, data []byte) error {
	part, err := m.w.CreateFormFile(fieldName, filepath.Base(fileName))
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// writeFileFullPath writes a file part with the full path used as the
// Content-Disposition filename. This is used for batch uploads where the
// server parses the destination path from the filename directly.
func (m *multipartFileWriter) writeFileFullPath(fieldName, fullPath string, data []byte) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fieldName), escapeQuotes(fullPath)))
	h.Set("Content-Type", "application/octet-stream")
	part, err := m.w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// escapeQuotes escapes backslashes and double quotes for use inside a
// Content-Disposition header value.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (m *multipartFileWriter) close() error {
	return m.w.Close()
}
