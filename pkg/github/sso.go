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

	ghgraphql "github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type TestGitHubOrg struct {
	users map[string]user
	Org   string
}

type user struct {
	email string
	saml  string
}

var query struct {
	Viewer struct {
		Login     ghgraphql.String
		CreatedAt ghgraphql.DateTime
	}
}

func GetSSOInfo(ctx context.Context, s *StaticTokenSource, endpoint string) *ghgraphql.Client {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: s.GetStaticToken(),
	}))

	var gqClient *ghgraphql.Client
	if endpoint != DefaultGitHubEndpointURL {
		gqClient = ghgraphql.NewEnterpriseClient(endpoint, httpClient)
	} else {
		gqClient = ghgraphql.NewClient(httpClient)
	}

	err := gqClient.Query(context.Background(), &query, nil)
	if err != nil {
		fmt.Println("Failed to query")
	}
	fmt.Println("    Login:", query.Viewer.Login)
	fmt.Println("CreatedAt:", query.Viewer.CreatedAt)
	tgo := &TestGitHubOrg{
		Org: "abcxyz",
	}
	err = tgo.saml(ctx, gqClient)
	if err != nil {
		fmt.Println("oops: %w", err)
		return nil
	}
	return gqClient
}

// saml finds all the SAML identities in the GitHub organization.
func (g *TestGitHubOrg) saml(ctx context.Context, client *ghgraphql.Client) error {
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
						EndCursor   ghgraphql.String
						HasNextPage bool
					}
				} `graphql:"externalIdentities(first: 100, after: $cursor)"`
			}
		} `graphql:"organization(login: $org)"`
	}
	vars := map[string]any{
		"org":    ghgraphql.String(g.Org),
		"cursor": (*ghgraphql.String)(nil),
	}

	for {
		if err := client.Query(ctx, &samlQuery, vars); err != nil {
			fmt.Println("executing GraphQL query: %w", err)
			return fmt.Errorf("executing GraphQL query: %w", err)
		}
		for _, edge := range samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.Edges {
			g.users[edge.Node.User.Login] = user{g.users[edge.Node.User.Login].email, edge.Node.SAMLIdentity.NameID}
		}
		if !samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.HasNextPage {
			break
		}
		vars["cursor"] = ghgraphql.NewString(samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.EndCursor)
	}
	fmt.Println(g.users)

	return nil
}
