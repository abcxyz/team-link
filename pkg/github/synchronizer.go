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

	"github.com/google/go-github/v56/github"
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
	ghClient := s.client.WithAuthToken(t)

	// Get current team members' username from GitHub and expected team members'
	// user from the team object.
	gotActiveMemberUsernames, err := s.curTeamUsernames(ctx, ghClient, team.OrgId, team.TeamId, s.listActiveTeamMembersWithRetry)
	if err != nil {
		return fmt.Errorf("failed to get active GitHub team members: %w", err)
	}
	gotPendingInvitationUsernames, err := s.curTeamUsernames(ctx, ghClient, team.OrgId, team.TeamId, s.listPendingTeamInvitationsWithRetry)
	if err != nil {
		return fmt.Errorf("failed to get pending GitHub team invitations: %w", err)
	}
	gotUsernames := append(gotActiveMemberUsernames, gotPendingInvitationUsernames...)
	wantUsernames := usernames(team)

	var retErr error
	// Add GitHub team memberships.
	for _, u := range sets.Subtract(wantUsernames, gotUsernames) {
		if err := s.addTeamMembersWithRetry(ctx, ghClient, team.OrgId, team.TeamId, u); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	// Remove GitHub team memberships.
	for _, u := range sets.Subtract(gotUsernames, wantUsernames) {
		if err := s.removeTeamMembersWithRetry(ctx, ghClient, team.OrgId, team.TeamId, u); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	return retErr
}

// teamUsernames returns a list of GitHub usernames that are members or has
// invitations to the given Github team.
func (s *Synchronizer) curTeamUsernames(
	ctx context.Context,
	c *github.Client,
	orgID, teamID int64,
	f func(ctx context.Context, c *github.Client, orgID, teamID int64, opt *github.ListOptions) ([]string, *github.Response, error),
) ([]string, error) {
	var res []string
	opt := &github.ListOptions{
		PerPage: 100,
	}
	for {
		usernames, resp, err := f(ctx, c, orgID, teamID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to get team member usernames: %w", err)
		}

		res = append(res, usernames...)
		if resp.NextPage == 0 {
			break // No more pages to fetch
		}
		opt.Page = resp.NextPage
	}
	return res, nil
}

func (s *Synchronizer) addTeamMembersWithRetry(ctx context.Context, c *github.Client, orgID, teamID int64, username string) error {
	return retry.Do(ctx, s.retry, func(ctx context.Context) error { //nolint:wrapcheck
		_, _, err := c.Teams.AddTeamMembershipByID(ctx, orgID, teamID, username, &github.TeamAddTeamMembershipOptions{Role: "member"})
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to add GitHub team members: %w", err))
		}
		return nil
	})
}

func (s *Synchronizer) removeTeamMembersWithRetry(ctx context.Context, c *github.Client, orgID, teamID int64, username string) error {
	return retry.Do(ctx, s.retry, func(ctx context.Context) error { //nolint:wrapcheck
		_, err := c.Teams.RemoveTeamMembershipByID(ctx, orgID, teamID, username)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to remove GitHub team members: %w", err))
		}
		return nil
	})
}

func (s *Synchronizer) listActiveTeamMembersWithRetry(ctx context.Context, c *github.Client, orgID, teamID int64, opt *github.ListOptions) (usernames []string, resp *github.Response, retErr error) {
	var members []*github.User
	o := &github.TeamListTeamMembersOptions{
		Role:        "all",
		ListOptions: *opt,
	}
	retErr = retry.Do(ctx, s.retry, func(ctx context.Context) error {
		var err error
		members, resp, err = c.Teams.ListTeamMembersByID(ctx, orgID, teamID, o)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to list team membership: %w", err))
		}
		return nil
	})
	for _, m := range members {
		usernames = append(usernames, *m.Login)
	}
	return usernames, resp, retErr //nolint:wrapcheck
}

func (s *Synchronizer) listPendingTeamInvitationsWithRetry(ctx context.Context, c *github.Client, orgID, teamID int64, opt *github.ListOptions) (usernames []string, resp *github.Response, retErr error) {
	var invitations []*github.Invitation
	retErr = retry.Do(ctx, s.retry, func(ctx context.Context) error {
		var err error
		invitations, resp, err = c.Teams.ListPendingTeamInvitationsByID(ctx, orgID, teamID, opt)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to list team invitations: %w", err))
		}
		return nil
	})
	for _, i := range invitations {
		usernames = append(usernames, *i.Login)
	}
	return usernames, resp, retErr //nolint:wrapcheck
}

func (s *Synchronizer) getAccessToken(ctx context.Context) (string, error) {
	tr := &githubauth.TokenRequestAllRepos{
		Permissions: map[string]string{
			"organization": "write",
		},
	}

	token, err := s.githubApp.AccessTokenAllRepos(ctx, tr)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	return token, nil
}

// usernames returns a list of GitHub usernames that are in the given
// team object.
func usernames(team *v1alpha1.GitHubTeam) []string {
	res := make([]string, len(team.Users))
	for i, m := range team.Users {
		res[i] = m.UserName // #nosec G601
	}
	return res
}
