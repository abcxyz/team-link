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

package gitlab

import (
	"context"
	"fmt"
	"net/http"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/abcxyz/team-link/pkg/github"
)

// ClientProvider provides a GitLab client.
type ClientProvider struct {
	httpClient  *http.Client
	instanceURL string
	keyProvider github.KeyProvider
}

// NewGitLabClientProvider creates a new GitLabClientProvider.
func NewGitLabClientProvider(httpClient *http.Client, instanceURL string, keyProvider github.KeyProvider) *ClientProvider {
	return &ClientProvider{
		httpClient:  httpClient,
		instanceURL: instanceURL,
		keyProvider: keyProvider,
	}
}

// Client returns a GitLab client initialized with a PAT.
func (g *ClientProvider) Client(ctx context.Context) (*gitlab.Client, error) {
	token, err := g.keyProvider.Key(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitLab token: %w", err)
	}
	gitlabClient, err := gitlab.NewClient(string(token), gitlab.WithHTTPClient(g.httpClient), gitlab.WithBaseURL(g.instanceURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}
	return gitlabClient, nil
}
