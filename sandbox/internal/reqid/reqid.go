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

// Package reqid provides helpers for propagating a request ID via context.
//
// The request ID is used for request-chain tracing across SUFY API calls.
// The SDK automatically extracts the request ID from the Context and sets it
// as the X-Reqid HTTP request header.
//
// Example:
//
//	ctx := reqid.WithReqid(ctx, "my-request-id")
//	// Subsequent API calls using ctx will carry the X-Reqid header.
//
//	id, ok := reqid.ReqidFromContext(ctx)
package reqid

import (
	"context"
)

type reqidKey struct{}

// WithReqid attaches the given reqid to the context.
func WithReqid(ctx context.Context, reqid string) context.Context {
	return context.WithValue(ctx, reqidKey{}, reqid)
}

// ReqidFromContext retrieves the reqid stored in the context, if any.
func ReqidFromContext(ctx context.Context) (reqid string, ok bool) {
	reqid, ok = ctx.Value(reqidKey{}).(string)
	return
}
