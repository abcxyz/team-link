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
	"strconv"

	"github.com/abcxyz/pkg/githubauth"
)

// KeyProvider provides a private key.
type KeyProvider interface {
	Key(ctx context.Context) ([]byte, error)
}

type AppTokenSource struct {
	keyProvider KeyProvider
	appID       string
}

func NewAppTokenSource(keyProvider KeyProvider, appID string) *AppTokenSource {
	return &AppTokenSource{
		keyProvider: keyProvider,
		appID:       appID,
	}
}

func (s *AppTokenSource) TokenForOrg(ctx context.Context, orgID int64) (string, error) {
	// TODO(https://github.com/abcxyz/team-link/issues/45): Consider caching the tokens we mint in this method.
	privateKey, err := s.keyProvider.Key(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get GitHub app private key: %w", err)
	}
	app, err := githubauth.NewApp(
		s.appID,
		string(privateKey),
	)
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
