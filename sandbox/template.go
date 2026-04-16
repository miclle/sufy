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
	"time"
)

// ListTemplates lists all templates visible to the caller.
func (c *Client) ListTemplates(ctx context.Context, __xgo_optional_params *ListTemplatesParams) ([]Template, error) {
	resp, err := c.api.ListTemplatesWithResponse(ctx, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templatesFromAPI(*resp.JSON200), nil
}

// CreateTemplate creates a new template (v3 API).
func (c *Client) CreateTemplate(ctx context.Context, body CreateTemplateParams) (*TemplateCreateResponse, error) {
	resp, err := c.api.CreateTemplateV3WithResponse(ctx, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON202 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateCreateResponseFromAPI(resp.JSON202), nil
}

// GetTemplate returns template details together with its build history.
func (c *Client) GetTemplate(ctx context.Context, templateID string, __xgo_optional_params *GetTemplateParams) (*TemplateWithBuilds, error) {
	resp, err := c.api.GetTemplateWithResponse(ctx, templateID, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateWithBuildsFromAPI(resp.JSON200), nil
}

// DeleteTemplate deletes a template.
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) error {
	resp, err := c.api.DeleteTemplateWithResponse(ctx, templateID)
	if err != nil {
		return err
	}
	sc := resp.HTTPResponse.StatusCode
	if sc != 200 && sc != 204 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// UpdateTemplate updates a template.
func (c *Client) UpdateTemplate(ctx context.Context, templateID string, body UpdateTemplateParams) error {
	resp, err := c.api.UpdateTemplateWithResponse(ctx, templateID, body.toAPI())
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// GetTemplateBuildStatus returns the status of a template build.
func (c *Client) GetTemplateBuildStatus(ctx context.Context, templateID, buildID string, __xgo_optional_params *GetBuildStatusParams) (*TemplateBuildInfo, error) {
	resp, err := c.api.GetTemplateBuildStatusWithResponse(ctx, templateID, buildID, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateBuildInfoFromAPI(resp.JSON200), nil
}

// GetTemplateBuildLogs returns the build logs for a template build.
func (c *Client) GetTemplateBuildLogs(ctx context.Context, templateID, buildID string, __xgo_optional_params *GetBuildLogsParams) (*TemplateBuildLogs, error) {
	resp, err := c.api.GetTemplateBuildLogsWithResponse(ctx, templateID, buildID, __xgo_optional_params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateBuildLogsFromAPI(resp.JSON200), nil
}

// StartTemplateBuild starts a template build (v2 API).
func (c *Client) StartTemplateBuild(ctx context.Context, templateID, buildID string, body StartTemplateBuildParams) error {
	apiBody, err := body.toAPI()
	if err != nil {
		return err
	}
	resp, err := c.api.StartTemplateBuildV2WithResponse(ctx, templateID, buildID, apiBody)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 202 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// GetTemplateFiles returns an upload URL for a template build file.
func (c *Client) GetTemplateFiles(ctx context.Context, templateID, hash string) (*TemplateBuildFileUpload, error) {
	resp, err := c.api.GetTemplateFilesWithResponse(ctx, templateID, hash)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateBuildFileUploadFromAPI(resp.JSON201), nil
}

// GetTemplateByAlias resolves a template alias.
func (c *Client) GetTemplateByAlias(ctx context.Context, alias string) (*TemplateAliasResponse, error) {
	resp, err := c.api.GetTemplateByAliasWithResponse(ctx, alias)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return templateAliasResponseFromAPI(resp.JSON200), nil
}

// AssignTemplateTags assigns tags to a template build.
func (c *Client) AssignTemplateTags(ctx context.Context, body ManageTagsParams) (*AssignedTemplateTags, error) {
	resp, err := c.api.AssignTemplateTagsWithResponse(ctx, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return assignedTemplateTagsFromAPI(resp.JSON201), nil
}

// DeleteTemplateTags removes tags from a template.
func (c *Client) DeleteTemplateTags(ctx context.Context, body DeleteTagsParams) error {
	resp, err := c.api.DeleteTemplateTagsWithResponse(ctx, body.toAPI())
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// WaitForBuild polls GetTemplateBuildStatus until the build reaches a terminal
// state ("ready" or "error"). The default poll interval is 2 seconds and can be
// customized with WithPollInterval and related options.
func (c *Client) WaitForBuild(ctx context.Context, templateID, buildID string, opts ...PollOption) (*TemplateBuildInfo, error) {
	o := defaultPollOpts(2 * time.Second)
	for _, fn := range opts {
		fn(o)
	}

	return pollLoop(ctx, o, func() (bool, *TemplateBuildInfo, error) {
		info, err := c.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
		if err != nil {
			return false, nil, fmt.Errorf("get build status %s/%s: %w", templateID, buildID, err)
		}
		switch info.Status {
		case BuildStatusReady:
			return true, info, nil
		case BuildStatusError:
			return true, info, fmt.Errorf("build %s/%s failed", templateID, buildID)
		}
		return false, nil, nil
	})
}
