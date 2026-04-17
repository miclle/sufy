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
	"net/http"
)

// ListInjectionRules returns all injection rules that belong to the caller.
func (c *Client) ListInjectionRules(ctx context.Context) ([]InjectionRule, error) {
	resp, err := c.api.GetInjectionRulesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return injectionRulesFromAPI(*resp.JSON200)
}

// CreateInjectionRule creates a new saved injection rule.
func (c *Client) CreateInjectionRule(ctx context.Context, params CreateInjectionRuleParams) (*InjectionRule, error) {
	body, err := params.toAPI()
	if err != nil {
		return nil, err
	}
	resp, err := c.api.PostInjectionRulesWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r, err := injectionRuleFromAPI(*resp.JSON201)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetInjectionRule returns the injection rule with the given ID.
func (c *Client) GetInjectionRule(ctx context.Context, ruleID string) (*InjectionRule, error) {
	resp, err := c.api.GetInjectionRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r, err := injectionRuleFromAPI(*resp.JSON200)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateInjectionRule updates the injection rule with the given ID.
func (c *Client) UpdateInjectionRule(ctx context.Context, ruleID string, params UpdateInjectionRuleParams) (*InjectionRule, error) {
	body, err := params.toAPI()
	if err != nil {
		return nil, err
	}
	resp, err := c.api.PutInjectionRulesRuleIDWithResponse(ctx, ruleID, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r, err := injectionRuleFromAPI(*resp.JSON200)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteInjectionRule deletes the injection rule with the given ID.
func (c *Client) DeleteInjectionRule(ctx context.Context, ruleID string) error {
	resp, err := c.api.DeleteInjectionRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}
