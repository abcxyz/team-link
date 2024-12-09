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

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient creates a new NewGitHubClient.
func NewGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func NewGitHubTeamReadWriterWithAuthToken(ctx context.Context, token string) *TeamReadWriter {
	gts := NewGitHubTokenSource(token)
	client := NewGitHubClient(ctx, token)
	return NewTeamReadWriter(gts, client)
}
