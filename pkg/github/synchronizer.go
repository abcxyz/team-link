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

// package github defines the mechanism to update GitHub teams' memberships
// given a list of records of the expected team memberships.
package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v56/github"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

// Synchronizer that syncs github team memberships.
// TODO(#7): add retry strategy.
type Synchronizer struct {
	client    *github.Client
	githubApp *githubauth.App
}

// NewSynchronizer creates a new Synchronizer with provided clients and options.
func NewSynchronizer(ghClient *github.Client, ghApp *githubauth.App) *Synchronizer {
	return &Synchronizer{
		client:    ghClient,
		githubApp: ghApp,
	}
}

// Sync overides several GitHub teams' memberships with the provided team
// membership snapshots.
// TODO(#3): populate the users' GitHub logins in the GitHubTeam object before
// this since they are required when updating GitHub team memberships.
func (s *Synchronizer) Sync(ctx context.Context, teams []*v1alpha1.GitHubTeam) error {
	// Configure Github auth token to the GitHub client.
	t, err := s.accessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	ghClient := s.client.WithAuthToken(t)

	var retErr error
	for _, team := range teams {
		// Get current team members' login from GitHub and expected team members'
		// user from the team object.
		gotActiveMemberLogins, err := s.currentTeamLogins(ctx, ghClient, team.GetOrgId(), team.GetTeamId(), listActiveTeamMembers)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get active GitHub team members for team(%d): %w", team.GetTeamId(), err))
			continue
		}
		gotPendingInvitationLogins, err := s.currentTeamLogins(ctx, ghClient, team.GetOrgId(), team.GetTeamId(), listPendingTeamInvitations)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get pending GitHub team invitations for team(%d): %w", team.GetTeamId(), err))
			continue
		}
		gotLogins := sets.Union(gotActiveMemberLogins, gotPendingInvitationLogins)
		wantLogins := loginsFromTeam(team)

		// Add GitHub team memberships.
		for _, u := range sets.Subtract(wantLogins, gotLogins) {
			if _, _, err := ghClient.Teams.AddTeamMembershipByID(ctx, team.GetOrgId(), team.GetTeamId(), u, &github.TeamAddTeamMembershipOptions{Role: "member"}); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to add GitHub team members for team(%d): %w", team.GetTeamId(), err))
			}
		}
		// Remove GitHub team memberships.
		for _, u := range sets.Subtract(gotLogins, wantLogins) {
			if _, err := ghClient.Teams.RemoveTeamMembershipByID(ctx, team.GetOrgId(), team.GetTeamId(), u); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to remove GitHub team members for team(%d): %w", team.GetTeamId(), err))
			}
		}
	}
	return retErr
}

// currentTeamLogins returns a list of GitHub logins that are members or has
// invitations to the given GitHub team.
// TODO(#6): refactor the paginated GitHub API call.
func (s *Synchronizer) currentTeamLogins(
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
		logins, resp, err := f(ctx, c, orgID, teamID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to get team member logins: %w", err)
		}

		res = append(res, logins...)
		if resp.NextPage == 0 {
			break // No more pages to fetch
		}
		opt.Page = resp.NextPage
	}
	return res, nil
}

func listActiveTeamMembers(ctx context.Context, c *github.Client, orgID, teamID int64, opt *github.ListOptions) ([]string, *github.Response, error) {
	o := &github.TeamListTeamMembersOptions{
		Role:        "all",
		ListOptions: *opt,
	}
	members, resp, err := c.Teams.ListTeamMembersByID(ctx, orgID, teamID, o)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list team membership: %w", err)
	}
	logins := make([]string, 0, len(members))
	for _, m := range members {
		logins = append(logins, *m.Login)
	}
	return logins, resp, nil
}

func listPendingTeamInvitations(ctx context.Context, c *github.Client, orgID, teamID int64, opt *github.ListOptions) ([]string, *github.Response, error) {
	invitations, resp, err := c.Teams.ListPendingTeamInvitationsByID(ctx, orgID, teamID, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list team invitations: %w", err)
	}
	logins := make([]string, 0, len(invitations))
	for _, inv := range invitations {
		logins = append(logins, *inv.Login)
	}
	return logins, resp, nil
}

func (s *Synchronizer) accessToken(ctx context.Context) (string, error) {
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

// loginsFromTeam returns a list of GitHub logins/usernames that are in the
// given team object.
func loginsFromTeam(team *v1alpha1.GitHubTeam) []string {
	res := make([]string, len(team.GetUsers()))
	for i, m := range team.GetUsers() {
		res[i] = m.GetLogin()
	}
	return res
}
