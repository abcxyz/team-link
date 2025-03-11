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

	"github.com/google/go-github/v61/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const DefaultGitHubEndpointURL = "https://github.com"

// NewTeamReadWriterWithStaticTokenSource creates a team readwriter using provided endpoint
// and static token source.
func NewTeamReadWriterWithStaticTokenSource(ctx context.Context, s *StaticTokenSource, endpoint string, orgTeamSSORequired map[int64]map[int64]bool) (*TeamReadWriter, error) {
	ghc := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: s.GetStaticToken(),
	})))
	var err error
	if endpoint != DefaultGitHubEndpointURL {
		if ghc, err = ghc.WithEnterpriseURLs(endpoint, endpoint); err != nil {
			return nil, fmt.Errorf("failed to create github client with enterprise endpoint %s: %w", endpoint, err)
		}
	}

	return NewTeamReadWriter(s, ghc, endpoint, orgTeamSSORequired), nil
}

// CreateGraphQLClientWithToken creates a graphQL client with a static token.
func CreateGraphQLClientWithToken(ctx context.Context, token, endpoint string) *githubv4.Client {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	}))
	var gqlClient *githubv4.Client
	if endpoint != DefaultGitHubEndpointURL {
		gqlClient = githubv4.NewEnterpriseClient(endpoint, httpClient)
	} else {
		gqlClient = githubv4.NewClient(httpClient)
	}
	return gqlClient
}
