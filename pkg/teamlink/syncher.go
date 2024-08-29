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

package teamlink

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/githubauth"
	api "github.com/abcxyz/team-link/apis/v1alpha2"
	tlgithub "github.com/abcxyz/team-link/pkg/github"
)

const cacheKey = "github-app-private-key"

// KeyProvider provides a private key.
type KeyProvider interface {
	Key(ctx context.Context) ([]byte, error)
}

type ConfiguredSyncer struct {
	keyProvider     KeyProvider
	httpClient      *http.Client
	githubClient    *github.Client
	privateKeyCache *cache.Cache[[]byte]
	githubAppID     string
	syncOpts        []tlgithub.Option
}

func NewConfiguredSyncer(
	keyProvider KeyProvider,
	httpClient *http.Client,
	githubClient *github.Client,
	cacheDuration time.Duration,
	githubAppID string,
	syncOpts ...tlgithub.Option,
) *ConfiguredSyncer {
	return &ConfiguredSyncer{
		keyProvider:     keyProvider,
		httpClient:      httpClient,
		githubClient:    githubClient,
		privateKeyCache: cache.New[[]byte](cacheDuration),
		githubAppID:     githubAppID,
		syncOpts:        syncOpts,
	}
}

// Get initializes a api.TeamSynchronizer and returns it.
func (r *ConfiguredSyncer) Get(ctx context.Context) (api.TeamSynchronizer, error) {
	privateKey, err := r.privateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get GitHub app private key: %w", err)
	}
	ghApp, err := githubauth.NewApp(
		r.githubAppID,
		string(privateKey),
		githubauth.WithHTTPClient(r.httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create github app: %w", err)
	}
	return tlgithub.NewSynchronizer(r.githubClient, ghApp, r.syncOpts...) //nolint:wrapcheck // Want passthrough
}

// privateKey gets GitHub app private key from the cache or the key provider.
func (r *ConfiguredSyncer) privateKey(ctx context.Context) ([]byte, error) {
	if privateKey, found := r.privateKeyCache.Lookup(cacheKey); found {
		return privateKey, nil
	}
	privateKey, err := r.keyProvider.Key(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get GitHub App private key: %w", err)
	}
	r.privateKeyCache.Set(cacheKey, privateKey)
	return privateKey, nil
}
