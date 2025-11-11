// Copyright 2024 Google LLC
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

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/team-link/v2/pkg/credentials"
)

// ClientProvider provides a GitLab client.
type ClientProvider struct {
	instanceURL string
	keyProvider credentials.KeyProvider
	httpClient  *http.Client
	clientOpts  []ClientOpt
}

type ClientOpt func(client *gitlab.Client)

// WithUserAgent sets the user agent on generated GitLab clients.
func WithUserAgent(userAgent string) ClientOpt {
	return func(client *gitlab.Client) {
		client.UserAgent = userAgent
	}
}

// NewGitLabClientProvider creates a new GitLabClientProvider.
func NewGitLabClientProvider(instanceURL string, keyProvider credentials.KeyProvider, httpClient *http.Client, opts ...ClientOpt) *ClientProvider {
	return &ClientProvider{
		instanceURL: instanceURL,
		keyProvider: keyProvider,
		httpClient:  httpClient,
		clientOpts:  opts,
	}
}

// Client returns a GitLab client initialized with a PAT.
func (g *ClientProvider) Client(ctx context.Context) (*gitlab.Client, error) {
	token, err := g.keyProvider.Key(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitLab token: %w", err)
	}

	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(g.instanceURL),
	}
	if g.httpClient != nil {
		opts = append(opts, gitlab.WithHTTPClient(g.httpClient))
	}
	gitlabClient, err := gitlab.NewClient(string(token), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}
	for _, opt := range g.clientOpts {
		opt(gitlabClient)
	}
	return gitlabClient, nil
}
