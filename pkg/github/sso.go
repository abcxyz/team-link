// Copyright 2024 The Authors (see AUTHORS file)
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
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v67/github"
	"github.com/shurcooL/githubv4"
)

// GetAllOrgsSAMLIdentities get all users that have SAML identities from each organization.
// This function returns a map with each orgID as key and a set of users with samkIdentities
// as value.
func GetAllOrgsSAMLIdentities(ctx context.Context, httpClient *http.Client, endpoint string, ghc *github.Client, orgTeamSSORequired map[int64]map[int64]bool) (map[int64]map[string]struct{}, error) {
	var gqlClient *githubv4.Client
	if endpoint != DefaultGitHubEndpointURL {
		gqlClient = githubv4.NewEnterpriseClient(endpoint, httpClient)
	} else {
		gqlClient = githubv4.NewClient(httpClient)
	}

	orgsSAMLMap := make(map[int64]map[string]struct{})

	for id := range orgTeamSSORequired {
		// GraphQL only supports query SAML using orgLogin.
		// We only know org ID, thus we need to get orgLogin info
		// before we run graphQL to get SAML info.
		org, _, err := ghc.Organizations.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get organization with id %d: %w", id, err)
		}
		res, err := GetOrgSAMLIdentities(ctx, gqlClient, *org.Login)
		if err != nil {
			return nil, fmt.Errorf("failed to get SAML info for org %s (org id: %d)", *org.Login, id)
		}
		orgsSAMLMap[id] = res
	}
	return orgsSAMLMap, nil
}

// GetOrgSAMLIdentitiesByOrgID get SAML identities for the github org.
// The return is a map with users have external SAML identity attached.
func GetOrgSAMLIdentitiesByOrgID(ctx context.Context, ghc *github.Client, gqc *githubv4.Client, orgID int64) (map[string]struct{}, error) {
	org, _, err := ghc.Organizations.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization with id %d: %w", orgID, err)
	}
	res, err := GetOrgSAMLIdentities(ctx, gqc, *org.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to get SAML info for org %s (org id: %d): %w", *org.Login, orgID, err)
	}
	return res, nil
}

// GetOrgSAMLIdentities get all users with SAML identities from the given org.
func GetOrgSAMLIdentities(ctx context.Context, client *githubv4.Client, orglogin string) (map[string]struct{}, error) {
	var samlQuery struct {
		Organization struct {
			SAMLIdentityProvider struct {
				ExternalIdentities struct {
					Edges []struct {
						Node struct {
							User struct {
								Login string
							}
							SAMLIdentity struct {
								NameID string
							}
						}
					}
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
				} `graphql:"externalIdentities(first: 100, after: $cursor)"`
			}
		} `graphql:"organization(login: $org)"`
	}
	vars := map[string]any{
		"org":    githubv4.String(orglogin),
		"cursor": (*githubv4.String)(nil),
	}

	orgSAMLMembers := make(map[string]struct{})

	for {
		if err := client.Query(ctx, &samlQuery, vars); err != nil {
			return nil, fmt.Errorf("executing GraphQL query: %w", err)
		}
		// We don't need to save the external SAML email nor check the externalSAML email domain,
		// this is because the above graphQL query only returns all users with external SAML identitys
		// in the given org. And each github org can only have sso.
		for _, edge := range samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.Edges {
			orgSAMLMembers[edge.Node.User.Login] = struct{}{}
		}
		if !samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.HasNextPage {
			break
		}
		vars["cursor"] = githubv4.NewString(samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.EndCursor)
	}

	return orgSAMLMembers, nil
}
