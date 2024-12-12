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
	"golang.org/x/oauth2"

	"github.com/abcxyz/pkg/cli"
)

const DefaultGitHubServerEndpoint = "https://github.com"

// ClientConfig is the config for github client.
type ClientConfig struct {
	Endpoint string
	Token    string
}

func (c *ClientConfig) RegisterFlags(set *cli.FlagSet) {
	f := set.NewSection("GITHUB OPTIONS")

	// The priority for parseing the flags are as follows.
	// It will use the value for toppest priority
	// 1. Read from input flags.
	// 2. Read from Envvars.
	// 3. Use default value.
	f.StringVar(&cli.StringVar{
		Name:    "github-server-endpoint",
		EnvVar:  "GITHUB_SERVER_URL",
		Target:  &c.Endpoint,
		Default: DefaultGitHubServerEndpoint,
		Usage:   `URL for github endpoint, example: "https://github.com"`,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-client-auth-token",
		Target: &c.Token,
		Usage:  `Token to authenticate with github`,
	})

	set.AfterParse(func(merr error) error {
		// In case user export GITHUB_SERVER_URL to empty string.
		if c.Endpoint == "" {
			c.Endpoint = DefaultGitHubServerEndpoint
		}
		return nil
	})
}

// NewGitHubClient create a github.Client base on ClientConfig.
func NewGitHubClient(ctx context.Context, c *ClientConfig) (*github.Client, error) {
	ghc := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: c.Token,
	})))
	var err error
	if c.Endpoint != DefaultGitHubServerEndpoint {
		if ghc, err = ghc.WithEnterpriseURLs(c.Endpoint, c.Endpoint); err != nil {
			return nil, fmt.Errorf("failed to create github client with enterprise endpoint %s: %w", c.Endpoint, err)
		}
	}
	return ghc, nil
}

// NewGitHubTeamReadWriter creates sa new ReadWriter for GitHub.
func NewGitHubTeamReadWriter(ctx context.Context, c *ClientConfig) (*TeamReadWriter, error) {
	gts := NewStaticTokenSource(c.Token)
	rw, err := NewGitHubClient(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHubReadWriter: %w", err)
	}
	return NewTeamReadWriter(gts, rw), nil
}
