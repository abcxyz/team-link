// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v67/github"
)

const ghesSCIMURLPath = "/api/v3/scim/v2/"

// SCIMClient handles direct HTTP communication with the GHES SCIM API.
// API doc: https://docs.github.com/en/enterprise-server@3.17/admin/managing-iam/provisioning-user-accounts-with-scim/provisioning-users-and-groups-with-scim-using-the-rest-api#provisioning-users-with-the-rest-api
type SCIMClient struct {
	httpClient *http.Client
	baseURL    *url.URL
}

// SCIMProvisionedIdentities represents the result of calling ListSCIMProvisionedIdentities.
// This is a copy/paste of the upstream SCIMProvisionedIdentities struct, with the Resources
// field typed as *SCIMUser instead of *SCIMUserAttributes.
// Temporary pending https://github.com/google/go-github/pull/3728.
type SCIMProvisionedIdentities struct {
	Schemas      []string    `json:"schemas,omitempty"`
	TotalResults *int        `json:"totalResults,omitempty"`
	ItemsPerPage *int        `json:"itemsPerPage,omitempty"`
	StartIndex   *int        `json:"startIndex,omitempty"`
	Resources    []*SCIMUser `json:"Resources,omitempty"`
}

// SCIMUser is a wrapper around the upstream SCIMUserAttributes struct that includes roles.
// Temporary pending https://github.com/google/go-github/pull/3728.
type SCIMUser struct {
	github.SCIMUserAttributes
	Roles []*SCIMUserRole `json:"roles,omitempty"`
}

// SCIMUserRole is an enterprise-wide role granted to the user. This is only
// supported in GitHub Enterprise Server, and not GitHub Enterprise Cloud.
// See the docs for allowed role names.
//
// https://docs.github.com/en/enterprise-server@latest/rest/enterprise-admin/scim?apiVersion=2022-11-28#provision-a-scim-enterprise-user
type SCIMUserRole struct {
	Value   string  `json:"value"`             // (Required.)
	Display *string `json:"display,omitempty"` // (Optional.)
	Type    *string `json:"type,omitempty"`    // (Optional.)
	Primary *bool   `json:"primary,omitempty"` // (Optional.)
}

// scimPatchOp represents a single SCIM patch operation.
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
type scimPatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

// scimPatchPayload is the body of a SCIM PATCH request.
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
type scimPatchPayload struct {
	Schemas    []string      `json:"schemas"`
	Operations []scimPatchOp `json:"Operations"`
}

// NewSCIMClient creates a new client for the GHES SCIM API.
func NewSCIMClient(httpClient *http.Client, baseURL string) (*SCIMClient, error) {
	u, err := url.Parse(strings.TrimSuffix(baseURL, "/") + ghesSCIMURLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url %q: %w", baseURL, err)
	}
	return &SCIMClient{httpClient: httpClient, baseURL: u}, nil
}

// ListUsers fetches all SCIM provisioned users from the enterprise, handling SCIM pagination.
func (c *SCIMClient) ListUsers(ctx context.Context) (map[string]*SCIMUser, error) {
	allUsers := make(map[string]*SCIMUser)
	startIndex := 1
	for {
		url := &url.URL{Path: "Users"}
		q := url.Query()
		q.Set("startIndex", strconv.Itoa(startIndex))
		q.Set("count", "100")
		url.RawQuery = q.Encode()

		var result SCIMProvisionedIdentities
		if _, err := c.do(ctx, http.MethodGet, c.baseURL.ResolveReference(url).String(), nil, &result); err != nil {
			return nil, fmt.Errorf("failed to list scim users starting at index %d: %w", startIndex, err)
		}

		if len(result.Resources) == 0 {
			break
		}
		for _, u := range result.Resources {
			allUsers[u.UserName] = u
		}
		if len(allUsers) >= *result.TotalResults {
			break
		}
		startIndex += len(result.Resources)
	}
	return allUsers, nil
}

// CreateUser provisions a new user.
func (c *SCIMClient) CreateUser(ctx context.Context, user *SCIMUser) (*SCIMUser, *github.Response, error) {
	path := "Users"
	// Schema for POST: https://datatracker.ietf.org/doc/html/rfc7644#section-3.3
	user.Schemas = append(user.Schemas, "urn:ietf:params:scim:schemas:core:2.0:User")
	user.Active = github.Bool(true)
	var createdUser SCIMUser
	resp, err := c.do(ctx, http.MethodPost, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), user, &createdUser)
	if err != nil {
		return nil, resp, err
	}
	return &createdUser, resp, err
}

// GetUser gets a SCIM provisioned user by their SCIM ID.
func (c *SCIMClient) GetUser(ctx context.Context, scimID string) (*SCIMUser, *github.Response, error) {
	path := fmt.Sprintf("Users/%s", scimID)
	var foundUser SCIMUser
	resp, err := c.do(ctx, http.MethodGet, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), nil, &foundUser)
	if err != nil {
		return nil, resp, err
	}
	return &foundUser, resp, err
}

// UpdateUser updates a user's attributes.
func (c *SCIMClient) UpdateUser(ctx context.Context, scimID string, user *SCIMUser) (*SCIMUser, *github.Response, error) {
	path := fmt.Sprintf("Users/%s", scimID)
	// Schema for PUT: https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.1
	user.Schemas = append(user.Schemas, "urn:ietf:params:scim:schemas:core:2.0:User")
	var updatedUser SCIMUser
	resp, err := c.do(ctx, http.MethodPut, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), user, &updatedUser)
	if err != nil {
		return nil, resp, err
	}
	return &updatedUser, resp, err
}

// DeactivateUser deactivates a user.
// https://docs.github.com/en/enterprise-server@3.17/admin/managing-iam/provisioning-user-accounts-with-scim/provisioning-users-and-groups-with-scim-using-the-rest-api#soft-deprovisioning-users-with-the-rest-api
func (c *SCIMClient) DeactivateUser(ctx context.Context, scimID string) (*SCIMUser, *github.Response, error) {
	path := fmt.Sprintf("Users/%s", scimID)
	// Schema for PATCH: https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
	payload := &scimPatchPayload{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		Operations: []scimPatchOp{
			{
				Op:    "replace",
				Value: map[string]bool{"active": false},
			},
		},
	}
	var deactivatedUser SCIMUser
	resp, err := c.do(ctx, http.MethodPatch, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), payload, &deactivatedUser)
	if err != nil {
		return nil, resp, err
	}
	return &deactivatedUser, resp, err
}

// ReactivateUser reinstating a suspended user.
// https://docs.github.com/en/enterprise-server@3.17/admin/managing-iam/provisioning-user-accounts-with-scim/deprovisioning-and-reinstating-users#reinstating-a-user-account-that-was-soft-deprovisioned
func (c *SCIMClient) ReactivateUser(ctx context.Context, scimID string) (*SCIMUser, *github.Response, error) {
	path := fmt.Sprintf("Users/%s", scimID)
	payload := &scimPatchPayload{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		Operations: []scimPatchOp{
			{
				Op:    "replace",
				Value: map[string]bool{"active": true},
			},
		},
	}
	var reactivatedUser SCIMUser
	resp, err := c.do(ctx, http.MethodPatch, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), payload, &reactivatedUser)
	if err != nil {
		return nil, resp, err
	}
	return &reactivatedUser, resp, err
}

// DeleteUser deactivates a user.
// https://docs.github.com/en/enterprise-server@3.17/admin/managing-iam/provisioning-user-accounts-with-scim/provisioning-users-and-groups-with-scim-using-the-rest-api#hard-deprovisioning-users-with-the-rest-api
func (c *SCIMClient) DeleteUser(ctx context.Context, scimID string) (*github.Response, error) {
	path := fmt.Sprintf("Users/%s", scimID)
	return c.do(ctx, http.MethodDelete, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), nil, nil)
}

// do is a helper to make a SCIM request.
func (c *SCIMClient) do(ctx context.Context, method, url string, body, result any) (*github.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// See headers in https://datatracker.ietf.org/doc/html/rfc7644
	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("Accept", "application/scim+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	ghResp := &github.Response{Response: resp}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ghResp, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return ghResp, fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return ghResp, nil
}
