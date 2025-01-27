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
	"strings"

	"github.com/shurcooL/githubv4"
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
		Login     githubv4.String
		CreatedAt githubv4.DateTime
	}
}

func GetSSOInfo(ctx context.Context, s *StaticTokenSource, endpoint string) *githubv4.Client {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: s.GetStaticToken(),
	}))

	var gqClient *githubv4.Client
	if endpoint != DefaultGitHubEndpointURL {
		gqClient = githubv4.NewEnterpriseClient(endpoint, httpClient)
	} else {
		gqClient = githubv4.NewClient(httpClient)
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
	// if err = tgo.testSAML(ctx, gqClient); err != nil {
	// 	fmt.Println("failed to test SAML: %w", err)
	// }
	// fmt.Println("-----testSAML called above------")
	// if err := tgo.findUsers(ctx, gqClient); err != nil {
	// 	fmt.Println("failed to find users: %w", err)
	// }
	// fmt.Println("-----findUsers called above------")
	if err = tgo.saml(ctx, gqClient); err != nil {
		fmt.Println("oops: %w", err)
		return nil
	}
	return gqClient
}

// saml finds all the SAML identities in the GitHub organization.
func (g *TestGitHubOrg) saml(ctx context.Context, client *githubv4.Client) error {
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
		"org":    githubv4.String(g.Org),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		if err := client.Query(ctx, &samlQuery, vars); err != nil {
			fmt.Println("executing GraphQL query: %w", err)
			return fmt.Errorf("executing GraphQL query: %w", err)
		}
		for _, edge := range samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.Edges {
			fmt.Println(edge.Node)
			fmt.Println(edge.Node.User)
			fmt.Println(edge.Node.SAMLIdentity)
			fmt.Println("-------hahahahahah--------")
			g.users[edge.Node.User.Login] = user{g.users[edge.Node.User.Login].email, edge.Node.SAMLIdentity.NameID}
		}
		if !samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.HasNextPage {
			fmt.Println("reach end of query")
			break
		}
		vars["cursor"] = githubv4.NewString(samlQuery.Organization.SAMLIdentityProvider.ExternalIdentities.PageInfo.EndCursor)
	}
	fmt.Println(g.users)

	if len(g.users) == 0 {
		return fmt.Errorf("no result returned from query")
	}

	return nil
}

func (g *TestGitHubOrg) findUsers(ctx context.Context, client *githubv4.Client) error {
	// client := githubv4.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: g.pat})))

	g.users = make(map[string]user)
	vars := map[string]any{
		"org":    githubv4.String(g.Org),
		"cursor": (*githubv4.String)(nil),
		"domain": githubv4.String("google.com"),
	}

	var userQuery struct {
		Organization struct {
			MembersWithRoles struct {
				Edges []struct {
					Node struct {
						Login                            string
						OrganizationVerifiedDomainEmails []string `graphql:"organizationVerifiedDomainEmails(login: $domain)"`
					}
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"membersWithRole(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $org)"`
	}

	for {
		err := client.Query(ctx, &userQuery, vars)
		if err != nil {
			return fmt.Errorf("executing GraphQL query: %w", err)
		}
		for _, edge := range userQuery.Organization.MembersWithRoles.Edges {
			fmt.Printf("Login: %s, Domain Emails: %s\n", edge.Node.Login, strings.Join(edge.Node.OrganizationVerifiedDomainEmails, "-"))
		}
		// 	g.users[edge.Node.Login] = user{g.Entity.Contact, ""}
		// 	for _, email := range edge.Node.OrganizationVerifiedDomainEmails {
		// 		if g.users[edge.Node.Login].email == g.Entity.Contact {
		// 			g.users[edge.Node.Login] = user{email, ""}
		// 		}
		// 		match, err := g.domainMatch(email)
		// 		if err != nil {
		// 			return err
		// 		}
		// 		if match {
		// 			g.users[edge.Node.Login] = user{email, ""}
		// 		}
		// 	}
		// }
		if !userQuery.Organization.MembersWithRoles.PageInfo.HasNextPage {
			break
		}
		vars["cursor"] = githubv4.NewString(userQuery.Organization.MembersWithRoles.PageInfo.EndCursor)
	}

	return nil
}

// func (g *TestGitHubOrg) testSAML(ctx context.Context, client *githubv4.Client) error {
// 	userLogin := "sailorlqh"
// 	orgLogin := "abcxyz"
// 	var query struct {
// 		Organization struct {
// 			SamlIdentityProvider struct {
// 				ExternalIdentities struct {
// 					Edges []struct {
// 						Node struct {
// 							SamlIdentity struct {
// 								NameId string `graphql:"nameId"`
// 							} `graphql:"samlIdentity"`
// 							User struct {
// 								Login string `graphql:"login"`
// 							} `graphql:"user"`
// 						} `graphql:"node"`
// 					} `graphql:"edges"`
// 				} `graphql:"externalIdentities(first: 100, login: $userLogin)"`
// 			} `graphql:"samlIdentityProvider"`
// 		} `graphql:"organization(login: $orgLogin)"`
// 	}

// 	variables := map[string]interface{}{
// 		"orgLogin":  githubv4.String(orgLogin),
// 		"userLogin": githubv4.String(userLogin),
// 	}

// 	err := client.Query(ctx, &query, variables)
// 	if err != nil {
// 		fmt.Println("failed to query")
// 		return fmt.Errorf("error querying GitHub GraphQL: %w", err)
// 	}

// 	fmt.Println(query)
// 	fmt.Printf("Organization: %s\n", query.Organization)
// 	fmt.Printf("SamlIdentityProvider: %s\n", query.Organization.SamlIdentityProvider)
// 	fmt.Printf("ExternalIdentities: %s\n", query.Organization.SamlIdentityProvider.ExternalIdentities)
// 	for _, edge := range query.Organization.SamlIdentityProvider.ExternalIdentities.Edges {
// 		fmt.Println(edge.Node.User.Login)
// 		if edge.Node.User.Login == userLogin {
// 			fmt.Println(edge.Node.User.Login)
// 			fmt.Println("found it")
// 		}
// 	}
// 	fmt.Printf("no SAML identity found for user %s in organization %s\n", userLogin, orgLogin)
// 	return fmt.Errorf("sailed to")
// 	return nil
// }
