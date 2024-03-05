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

// package github defines the mechanism to update GitHub team memberships given
// a record of the expected team memberships.
package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v55/github"
	"github.com/sethvargo/go-retry"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

// Synchronizer that syncs github team memberships.
type Synchronizer struct {
	client    *github.Client
	githubApp *githubauth.App
	// Optional retry backoff strategy, default is 5 attempts with fibonacci
	// backoff that starts at 500ms.
	retry retry.Backoff
}

// Option is the option to set up a Synchronizer.
type Option func(h *Synchronizer) *Synchronizer

// WithRetry provides retry strategy to the Synchronizer.
func WithRetry(b retry.Backoff) Option {
	return func(s *Synchronizer) *Synchronizer {
		s.retry = b
		return s
	}
}

// NewSynchronizer creates a new Synchronizer with provided clients and options.
func NewSynchronizer(ghClient *github.Client, ghApp *githubauth.App, opts ...Option) *Synchronizer {
	s := &Synchronizer{
		client:    ghClient,
		githubApp: ghApp,
	}
	for _, opt := range opts {
		s = opt(s)
	}
	if s.retry == nil {
		s.retry = retry.WithMaxRetries(5, retry.NewFibonacci(500*time.Millisecond))
	}
	return s
}

// Sync overides a GitHub team's memberships with the provided team membership
// snapshot.
// TODO(#3): populate the users' GitHub usernames in the GitHubTeam object
// before this since they are required when updating GitHub team memberships.
func (s *Synchronizer) Sync(ctx context.Context, team *v1alpha1.GitHubTeam) error {
	// Configure Github auth token to the GitHub client.
	t, err := s.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	s.client = s.client.WithAuthToken(t)

	// Get current team members' username from GitHub and expected team members'
	// user from the team object.
	gotUsernames, err := s.teamUsernames(ctx, team.OrgId, team.TeamId)
	if err != nil {
		return fmt.Errorf("failed to get GitHub team members: %w", err)
	}
	expectedUsernames := teamUsernames(team)

	var retErr error
	// Add GitHub team memberships.
	for _, u := range sets.Subtract(expectedUsernames, gotUsernames) {
		if err := s.addTeamMembersWithRetry(ctx, team.OrgId, team.TeamId, u); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	// Remove GitHub team memberships.
	for _, u := range sets.Subtract(gotUsernames, expectedUsernames) {
		if err := s.removeTeamMembersWithRetry(ctx, team.OrgId, team.TeamId, u); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	return retErr
}

// teamUsernames returns a list of GitHub usernames that are members to the
// given Github team.
func (s *Synchronizer) teamUsernames(ctx context.Context, orgID, teamID int64) ([]string, error) {
	var res []string
	opt := &github.TeamListTeamMembersOptions{
		Role:        "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		members, resp, err := s.listTeamMembersWithRetry(ctx, orgID, teamID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list team members: %w", err)
		}

		for _, m := range members {
			username := *m.Login
			res = append(res, username)
		}

		if resp.NextPage == 0 {
			break // No more pages to fetch
		}
		opt.Page = resp.NextPage
	}
	return res, nil
}

func (s *Synchronizer) addTeamMembersWithRetry(ctx context.Context, orgID, teamID int64, username string) error {
	return retry.Do(ctx, s.retry, func(ctx context.Context) error {
		_, _, err := s.client.Teams.AddTeamMembershipByID(ctx, orgID, teamID, username, &github.TeamAddTeamMembershipOptions{Role: "member"})
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to add GitHub team members: %w", err))
		}
		return nil
	}) //nolint:wrapcheck
}

func (s *Synchronizer) removeTeamMembersWithRetry(ctx context.Context, orgID, teamID int64, username string) error {
	return retry.Do(ctx, s.retry, func(ctx context.Context) error {
		_, err := s.client.Teams.RemoveTeamMembershipByID(ctx, orgID, teamID, username)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to remove GitHub team members: %w", err))
		}
		return nil
	}) //nolint:wrapcheck
}

func (s *Synchronizer) listTeamMembersWithRetry(ctx context.Context, orgID, teamID int64, opt *github.TeamListTeamMembersOptions) (members []*github.User, resp *github.Response, retErr error) {
	retErr = retry.Do(ctx, s.retry, func(ctx context.Context) error {
		var err error
		members, resp, err = s.client.Teams.ListTeamMembersByID(ctx, orgID, teamID, opt)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to list team membership: %w", err))
		}
		return nil
	})
	return members, resp, retErr
}

func (s *Synchronizer) getAccessToken(ctx context.Context) (string, error) {
	tr := &githubauth.TokenRequest{
		Repositories: []string{"all"},
		Permissions: map[string]string{
			"organization": "write",
			// "issues": "read",
		},
	}

	token, err := s.githubApp.AccessToken(ctx, tr)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	return token, nil
}

// teamUsernames returns a list of GitHub usernames that are in the given
// team object.
func teamUsernames(team *v1alpha1.GitHubTeam) []string {
	res := make([]string, len(team.Users))
	for i, m := range team.Users {
		j := i
		res[j] = m.UserName
	}
	return res
}
