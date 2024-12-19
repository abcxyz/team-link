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
	"os"
	"strconv"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/team-link/pkg/credentials"
)

// DefaultStaticTokenEnvVar is where we read default github token from.
const DefaultStaticTokenEnvVar = "TEAM_LINK_GITHUB_TOKEN" // #nosec G101

type AppTokenSource struct {
	keyProvider credentials.KeyProvider
	appID       string
	appOpts     []githubauth.Option
}

func NewAppTokenSource(keyProvider credentials.KeyProvider, appID string, appOpts ...githubauth.Option) *AppTokenSource {
	return &AppTokenSource{
		keyProvider: keyProvider,
		appID:       appID,
		appOpts:     appOpts,
	}
}

func (s *AppTokenSource) TokenForOrg(ctx context.Context, orgID int64) (string, error) {
	// TODO(https://github.com/abcxyz/team-link/issues/45): Consider caching the tokens we mint in this method.
	privateKey, err := s.keyProvider.Key(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get GitHub app private key: %w", err)
	}
	app, err := githubauth.NewApp(s.appID, privateKey, s.appOpts...)
	if err != nil {
		return "", fmt.Errorf("unable to create GitHub app: %w", err)
	}
	appInstallation, err := app.InstallationForOrg(ctx, strconv.FormatInt(orgID, 10))
	if err != nil {
		return "", fmt.Errorf("failed to get installation for org %d: %w", orgID, err)
	}
	token, err := appInstallation.AccessTokenAllRepos(ctx, &githubauth.TokenRequestAllRepos{
		Permissions: map[string]string{
			"members": "write",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get access token for org %d: %w", orgID, err)
	}
	return token, nil
}

// StaticTokenSource implements OrgTokenSource.
type StaticTokenSource struct {
	token string
}

func (s *StaticTokenSource) TokenForOrg(ctx context.Context, orgID int64) (string, error) {
	return s.token, nil
}

func (s *StaticTokenSource) GetStaticToken() string {
	return s.token
}

// NewStaticTokenSourceFromEnvVar creates a StaticTokenSource using token read from EnvVar.
func NewStaticTokenSourceFromEnvVar(envVarName string) (*StaticTokenSource, error) {
	if envVarName == "" {
		envVarName = DefaultStaticTokenEnvVar
	}
	token := os.Getenv(envVarName)
	if token == "" {
		return nil, fmt.Errorf("failed to get token from env var: %s", envVarName)
	}

	return &StaticTokenSource{
		token: token,
	}, nil
}
